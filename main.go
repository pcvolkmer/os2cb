package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocarina/gocsv"
	"golang.org/x/term"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"log"
	"os"
	"syscall"
	_ "syscall"
)

var (
	cli     *CLI
	context *kong.Context
	db      *sql.DB
)

type Globals struct {
	User      string   `short:"U" help:"Database username"`
	Password  string   `short:"P" help:"Database password"`
	Host      string   `short:"H" help:"Database host" default:"localhost"`
	Port      int      `help:"Database port" default:"3306"`
	Database  string   `short:"D" help:"Database name" default:"onkostar"`
	PatientId []string `help:"PatientenIDs der zu exportierenden Patienten. Kommagetrennt bei mehreren IDs"`
	Filename  string   `help:"Exportiere in diese Datei"`
	Append    bool     `help:"An bestehende Datei anhängen"`
	Csv       bool     `help:"Verwende CSV-Format anstelle TSV-Format. Trennung mit ';' für MS Excel" default:"false"`
}

type CLI struct {
	Globals

	ExportPatients struct {
	} `cmd:"" help:"Export patient data"`

	ExportSamples struct {
	} `cmd:"" help:"Export sample data"`
}

func init() {
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

	if len(cli.Password) == 0 {
		fmt.Print("Passwort: ")
		if bytePw, err := term.ReadPassword(int(syscall.Stdin)); err == nil {
			cli.Password = string(bytePw)
		}
		println()
	}

	if dbx, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cli.User, cli.Password, cli.Host, cli.Port, cli.Database)); err == nil {
		db = dbx
		defer db.Close()
	} else {
		return
	}

	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
		utf16bom := unicode.BOMOverride(win16be.NewEncoder())

		var writer *csv.Writer
		if cli.Csv {
			transformWriter := transform.NewWriter(out, utf16bom)
			writer = csv.NewWriter(transformWriter)
			writer.Comma = ';'
		} else {
			writer = csv.NewWriter(out)
			writer.Comma = '\t'
		}
		return gocsv.NewSafeCSVWriter(writer)
	})

	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
		utf16bom := unicode.BOMOverride(win16be.NewDecoder())

		var reader *csv.Reader
		if cli.Csv {
			transformReader := transform.NewReader(in, utf16bom)
			reader = csv.NewReader(transformReader)
			reader.Comma = ';'
		} else {
			reader = csv.NewReader(in)
			reader.Comma = '\t'
		}
		return reader
	})

	switch context.Command() {
	case "export-patients":
		handleCommand(cli, db, fetchAllPatientData)
	case "export-samples":
		handleCommand(cli, db, fetchAllSampleData)
	default:

	}

}

// Bearbeitet die Ausführung und ermittelt Daten abhängig von übergebener Funktion
func handleCommand[D PatientData | SampleData](cli *CLI, db *sql.DB, fetchFunc func(patientIds []string, db *sql.DB) ([]D, error)) {
	var result []D
	if cli.Append {
		if r, err := readFile(cli.Filename, result); err == nil {
			result = r
		} else {
			log.Fatalln(err.Error())
		}
	}

	if r, err := fetchFunc(cli.PatientId, db); err == nil {
		result = append(result, r...)
	} else {
		log.Fatalln(err.Error())
	}

	if err := writeFile(cli.Filename, result); err != nil {
		log.Fatalln(err.Error())
	}
}

// Ermittelt alle Patientendaten von allen angegebenen Patienten
func fetchAllPatientData(patientIds []string, db *sql.DB) ([]PatientData, error) {
	patients := InitPatients(db)
	result := []PatientData{}
	for _, patientId := range cli.PatientId {
		if data, err := patients.Fetch(patientId); err == nil {
			result = append(result, *data)
		} else {
			log.Println(err.Error())
		}
	}
	return result, nil
}

// Ermittelt alle Probendaten von allen angegebenen Patienten
func fetchAllSampleData(patientIds []string, db *sql.DB) ([]SampleData, error) {
	samples := InitSamples(db)
	result := []SampleData{}
	for _, patientId := range cli.PatientId {
		if data, err := samples.Fetch(patientId); err == nil {
			for _, d := range data {
				result = append(result, d)
			}
		} else {
			log.Println(err.Error())
		}
	}
	return result, nil
}

// Liest eine bestehende Datei ein
func readFile[D PatientData | SampleData](filename string, data []D) ([]D, error) {
	file, err := os.Open(cli.Filename)
	defer file.Close()
	if err != nil {
		return nil, errors.New("file: Datei kann nicht geöffnet werden")
	}
	if gocsv.UnmarshalFile(file, &data) != nil {
		return nil, errors.New("file: Datei kann nicht gelesen werden")
	}

	return data, nil
}

// Schreibt Daten in CSV/TSV Datei
func writeFile[D PatientData | SampleData](filename string, data []D) error {
	file, err := os.Create(cli.Filename)
	if err != nil {
		return errors.New("file: Datei kann nicht geöffnet werden")
	}

	if err := gocsv.MarshalFile(data, file); err != nil {
		return errors.New("file: In die Datei kann nicht geschrieben werden")
	}

	return nil
}
