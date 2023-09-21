package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io"
	"log"
	"strings"
	"syscall"
	_ "syscall"

	"github.com/alecthomas/kong"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocarina/gocsv"
	"golang.org/x/term"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	cli     *CLI
	context *kong.Context
	db      *sql.DB
)

type Globals struct {
	User      string   `short:"U" help:"Database username" required:""`
	Password  string   `short:"P" help:"Database password"`
	Host      string   `short:"H" help:"Database host" default:"localhost"`
	Port      int      `help:"Database port" default:"3306"`
	Ssl       string   `help:"SSL-Verbindung ('true', 'false', 'skip-verify', 'preferred')" default:"false"`
	Database  string   `short:"D" help:"Database name" default:"onkostar"`
	PatientID []string `help:"PatientenIDs der zu exportierenden Patienten. Kommagetrennt bei mehreren IDs"`
	IDPrefix  string   `help:"Zu verwendender Prefix für anonymisierte IDs. 'WUE', wenn nicht anders angegeben." default:"WUE"`
	AllTk     bool     `help:"Diagnosen: Erlaube Diagnosen mit allen Tumorkonferenzen, nicht nur Diagnosen mit MTBs"`
	MtbType   string   `help:"MTB-Typ der Tumorkonferenz in Onkostar. Wenn nicht angegeben, Wert: '27'" default:"27"`
	NoAnon    bool     `help:"Keine ID-Anonymisierung anwenden. Hierbei wird auch das ID-Prefix ignoriert."`
}

type CLI struct {
	Globals

	ExportPatients struct {
		Filename string `help:"Exportiere in diese Datei" required:""`
		Append   bool   `help:"An bestehende Datei anhängen" default:"false"`
		Csv      bool   `help:"Verwende CSV-Format anstelle TSV-Format. Trennung mit ';' für MS Excel" default:"false"`
	} `cmd:"" help:"Export patient data"`

	ExportSamples struct {
		Filename string `help:"Exportiere in diese Datei" required:""`
		Append   bool   `help:"An bestehende Datei anhängen" default:"false"`
		Csv      bool   `help:"Verwende CSV-Format anstelle TSV-Format. Trennung mit ';' für MS Excel" default:"false"`
	} `cmd:"" help:"Export sample data"`

	Preview struct {
	} `cmd:"" help:"Show patient data. Exit Preview-Mode with <CTRL>+'C'"`
}

func initCLI() {
	cli = &CLI{
		Globals: Globals{},
	}
	context = kong.Parse(cli,
		kong.Name("os2cb"),
		kong.Description("A simple tool to export data from Onkostar into TSV file format for cBioportal"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
}

func main() {

	initCLI()

	if len(cli.Password) == 0 {
		fmt.Print("Passwort: ")
		if bytePw, err := term.ReadPassword(int(syscall.Stdin)); err == nil {
			cli.Password = string(bytePw)
		}
		println()
	}

	dbCfg := mysql.Config{
		User:                 cli.User,
		Passwd:               cli.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", cli.Host, cli.Port),
		DBName:               cli.Database,
		AllowNativePasswords: true,
		TLSConfig:            cli.Ssl,
	}

	if dbx, err := sql.Open("mysql", dbCfg.FormatDSN()); err == nil {
		if err := dbx.Ping(); err == nil {
			db = dbx
			defer func(db *sql.DB) {
				err := db.Close()
				if err != nil {
					log.Println("Cannot close database connection")
				}
			}(db)
		} else {
			log.Fatalf("Cannot connect to Database: %s\n", err.Error())
		}
	} else {
		log.Fatalf("Cannot connect to Database: %s\n", err.Error())
	}

	gocsv.SetCSVWriter(getCsvWriter(cli.ExportPatients.Csv || cli.ExportSamples.Csv))
	gocsv.SetCSVReader(getCsvReader(cli.ExportPatients.Csv || cli.ExportSamples.Csv))

	switch context.Command() {
	case "export-patients":
		handleCommand(cli, db, FetchAllPatientData)
	case "export-samples":
		handleCommand(cli, db, FetchAllSampleData)
	case "preview":
		preview(db)
	default:

	}

}

func AnonymizedID(id string) string {
	if cli.NoAnon {
		return id
	}

	sha := sha256.New()
	sha.Write([]byte(id))
	hash := hex.EncodeToString(sha.Sum(nil))

	return cli.IDPrefix + "_" + hash[0:10]
}

// Übergibt Methode zum Erstellen des passenden CsvWriters für TSV (cBioportal) oder CSV (Excel mit UTF16BE)
func getCsvWriter(isCsv bool) func(out io.Writer) *gocsv.SafeCSVWriter {
	return func(out io.Writer) *gocsv.SafeCSVWriter {
		win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
		utf16bom := unicode.BOMOverride(win16be.NewEncoder())

		var writer *csv.Writer
		if isCsv {
			transformWriter := transform.NewWriter(out, utf16bom)
			writer = csv.NewWriter(transformWriter)
			writer.Comma = ';'
		} else {
			writer = csv.NewWriter(out)
			writer.Comma = '\t'
		}
		return gocsv.NewSafeCSVWriter(writer)
	}
}

// Übergibt Methode zum Erstellen des passenden CsvReaders für TSV (cBioportal) oder CSV (Excel mit UTF16BE)
func getCsvReader(isCsv bool) func(in io.Reader) gocsv.CSVReader {
	return func(in io.Reader) gocsv.CSVReader {
		win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
		utf16bom := unicode.BOMOverride(win16be.NewDecoder())

		var reader *csv.Reader
		if isCsv {
			transformReader := transform.NewReader(in, utf16bom)
			reader = csv.NewReader(transformReader)
			reader.Comma = ';'
		} else {
			reader = csv.NewReader(in)
			reader.Comma = '\t'
		}
		reader.Comment = '#'
		return reader
	}
}

// Bearbeitet die Ausführung und ermittelt Daten abhängig von übergebener Funktion
func handleCommand[D PatientData | SampleData](cli *CLI, db *sql.DB, fetchFunc func(patientIds []string, db *sql.DB) ([]D, error)) {
	var result []D
	var filename string
	if len(cli.ExportPatients.Filename) > 0 {
		filename = cli.ExportPatients.Filename
	} else if len(cli.ExportSamples.Filename) > 0 {
		filename = cli.ExportSamples.Filename
	}

	if cli.ExportPatients.Append || cli.ExportSamples.Append {
		if r, err := ReadFile(filename, result); err == nil {
			result = r
		} else {
			log.Fatalln(err.Error())
		}
	}

	if r, err := fetchFunc(cli.PatientID, db); err == nil {
		result = append(result, r...)
	} else {
		log.Fatalln(err.Error())
	}

	if err := WriteFile(filename, result); err != nil {
		log.Fatalln(err.Error())
	}
}

func preview(db *sql.DB) {
	NewBrowser(cli.PatientID, db).Show()
}

// Ermittelt alle Patientendaten von allen angegebenen Patienten
func FetchAllPatientData(patientIds []string, db *sql.DB) ([]PatientData, error) {
	patients := InitPatients(db)
	var result []PatientData
	for _, patientID := range patientIds {
		if data, err := patients.Fetch(patientID, cli.MtbType, cli.AllTk); err == nil {
			result = append(result, *data)
		} else {
			if !strings.HasPrefix(context.Command(), "display") {
				log.Println(err.Error())
			}
		}
	}
	return result, nil
}

// Ermittelt alle Probendaten von allen angegebenen Patienten
func FetchAllSampleData(patientIds []string, db *sql.DB) ([]SampleData, error) {
	samples := InitSamples(db)
	var result []SampleData
	for _, patientID := range patientIds {
		if data, err := samples.Fetch(patientID); err == nil {
			for _, d := range data {
				result = append(result, d)
			}
		} else {
			if !strings.HasPrefix(context.Command(), "display") {
				log.Println(err.Error())
			}
		}
	}
	return result, nil
}
