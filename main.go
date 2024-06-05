package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/go-sql-driver/mysql"
	"github.com/gocarina/gocsv"
	"golang.org/x/term"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"log"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	_ "syscall"
)

var (
	cli     *CLI
	context *kong.Context
	db      *sql.DB
)

type Globals struct {
	User     string `short:"U" help:"Database username" required:""`
	Password string `short:"P" help:"Database password"`
	Host     string `short:"H" help:"Database host" default:"localhost"`
	Port     int    `help:"Database port" default:"3306"`
	Ssl      string `help:"SSL-Verbindung ('true', 'false', 'skip-verify', 'preferred')" default:"false"`
	Database string `short:"D" help:"Database name" default:"onkostar"`
	IDPrefix string `help:"Zu verwendender Prefix für anonymisierte IDs. 'WUE', wenn nicht anders angegeben." default:"WUE"`
	AllTk    bool   `help:"Diagnosen: Erlaube Diagnosen mit allen Tumorkonferenzen, nicht nur Diagnosen mit MTBs"`
	MtbType  string `help:"MTB-Typ der Tumorkonferenz in Onkostar. Wenn nicht angegeben, Wert: '27'" default:"27"`
	NoAnon   bool   `help:"Keine ID-Anonymisierung anwenden. Hierbei wird auch das ID-Prefix ignoriert."`
}

type PatientSelection struct {
	PatientID []string `help:"PatientenIDs der zu exportierenden Patienten. Kommagetrennt bei mehreren IDs" group:"Patienten" xor:"PatientID,OcaPlus" required:"true"`
	OcaPlus   bool     `help:"Alle Patienten mit OCAPlus-Panel" group:"Patienten" xor:"PatientID,OcaPlus" required:"true"`
	PersStamm int      `help:"ID des Personenstamms" group:"Patienten" default:"4"`
}

type CLI struct {
	Globals
	PatientSelection

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

	ExportXlsx struct {
		Filename string `help:"Exportiere in diese Datei" required:""`
	} `aliases:"export-xls" cmd:"" help:"Export all into Excel-File"`

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

	if cli.OcaPlus {
		patients := InitPatients(db)
		cli.PatientID, _ = patients.FetchOcaPlusPatientIds()
	}

	switch context.Command() {
	case "export-patients":
		handleCommand(cli, db, FetchAllPatientData)
	case "export-samples":
		handleCommand(cli, db, FetchAllSampleData)
	case "export-xlsx":
		exportXlsx(cli, cli.PatientID, db)
	case "export-xls":
		exportXlsx(cli, cli.PatientID, db)
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

func exportXlsx(cli *CLI, patientIds []string, db *sql.DB) {
	if !strings.HasSuffix(cli.ExportXlsx.Filename, ".xlsx") {
		log.Fatalf("Cannot use filename: '%s'. Required filename suffix is '.xlsx'", cli.ExportXlsx.Filename)
		return
	}

	patientsData := make([]PatientData, 0)
	samplesData := make([]SampleData, 0)
	if data, err := FetchAllPatientData(patientIds, db); err == nil {
		patientsData = append(patientsData, data...)
	} else {
		log.Printf(err.Error())
	}
	if data, err := FetchAllSampleData(patientIds, db); err == nil {
		samplesData = append(samplesData, data...)
	} else {
		log.Printf(err.Error())
	}

	if err := WriteXlsxFile(cli.ExportXlsx.Filename, patientsData, samplesData); err != nil {
		log.Fatalln(err.Error())
	}
}

func preview(db *sql.DB) {
	NewBrowser(cli.PatientID, cli.NoAnon, db).Show()
}

// Ermittelt alle Patientendaten von allen angegebenen Patienten
func FetchAllPatientData(patientIds []string, db *sql.DB) ([]PatientData, error) {
	patients := InitPatients(db)
	if data, err := patients.FetchBy(patientIds, cli.MtbType, cli.AllTk); err == nil {
		return data, nil
	} else {
		if !strings.HasPrefix(context.Command(), "preview") {
			log.Println(err.Error())
		}
	}
	return []PatientData{}, nil
}

// Ermittelt alle Probendaten von allen angegebenen Patienten
func FetchAllSampleData(patientIds []string, db *sql.DB) ([]SampleData, error) {
	samples := InitSamples(db, cli.OcaPlus)
	var result []SampleData
	for _, patientID := range patientIds {
		if data, err := samples.Fetch(patientID); err == nil {
			for _, d := range data {
				result = append(result, d)
			}
		} else {
			if !strings.HasPrefix(context.Command(), "preview") {
				log.Println(err.Error())
			}
		}
	}
	return result, nil
}
