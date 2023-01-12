package main

import (
	"errors"
	"github.com/gocarina/gocsv"
	"log"
	"os"
)

// Liest eine bestehende Datei ein
func ReadFile[D PatientData | SampleData](filename string, data []D) ([]D, error) {
	file, err := os.Open(filename)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println("Cannot close file")
		}
	}(file)
	if err != nil {
		return nil, errors.New("file: Datei kann nicht geöffnet werden")
	}
	if gocsv.UnmarshalFile(file, &data) != nil {
		return nil, errors.New("file: Datei kann nicht gelesen werden")
	}

	return data, nil
}

// Schreibt Daten in CSV/TSV Datei
func WriteFile[D PatientData | SampleData](filename string, data []D) error {
	file, err := os.Create(filename)
	if err != nil {
		return errors.New("file: Datei kann nicht geöffnet werden")
	}

	if err := gocsv.MarshalFile(data, file); err != nil {
		return errors.New("file: In die Datei kann nicht geschrieben werden")
	}

	return nil
}
