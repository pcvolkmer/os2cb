package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/alecthomas/kong"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocarina/gocsv"
	"golang.org/x/term"
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

	comma := '\t'
	if cli.Csv {
		comma = ';'
	}

	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		writer := csv.NewWriter(out)
		writer.Comma = comma
		return gocsv.NewSafeCSVWriter(writer)
	})

	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		reader := csv.NewReader(in)
		reader.Comma = comma
		return reader
	})

	switch context.Command() {
	case "export-patients":
		patients := InitPatients(db)
		result := []PatientData{}
		if cli.Append {
			file, err := os.Open(cli.Filename)
			defer file.Close()
			if err != nil {
				log.Fatalln("Datei kann nicht geöffnet werden")
			}
			if gocsv.UnmarshalFile(file, &result) != nil {
				log.Fatalln("Datei kann nicht gelesen werden")
			}
		}

		file, err := os.Create(cli.Filename)
		if err != nil {
			log.Fatalln("Datei kann nicht geöffnet werden")
		}

		for _, patientId := range cli.PatientId {
			if data, err := patients.Fetch(patientId); err == nil {
				result = append(result, *data)
			} else {
				log.Print(err.Error())
			}
		}

		if err := gocsv.MarshalFile(result, file); err != nil {
			log.Fatalln("In die Datei kann nicht geschrieben werden")
		}
	case "export-samples":
		patients := InitSamples(db)
		result := []SampleData{}
		if cli.Append {
			file, err := os.Open(cli.Filename)
			defer file.Close()
			if err != nil {
				log.Fatalln("Datei kann nicht geöffnet werden")
			}
			if gocsv.UnmarshalFile(file, &result) != nil {
				log.Fatalln("Datei kann nicht gelesen werden")
			}
		}

		file, err := os.Create(cli.Filename)
		if err != nil {
			log.Fatalln("Datei kann nicht geöffnet werden")
		}

		for _, patientId := range cli.PatientId {
			if data, err := patients.Fetch(patientId); err == nil {
				for _, d := range data {
					result = append(result, d)
				}
			} else {
				log.Print(err.Error())
			}
		}

		if err := gocsv.MarshalFile(result, file); err != nil {
			log.Fatalln("In die Datei kann nicht geschrieben werden")
		}
	default:

	}

}
