package main

import (
	_ "embed"
	"errors"
	"log"
	"os"
	"reflect"

	"github.com/gocarina/gocsv"
)

//go:embed resources/prefix-data_clinical_patient.txt
var prefixDataClinicalPatient string

//go:embed resources/prefix-data_clinical_sample.txt
var prefixDataClinicalSample string

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

	if output, err := gocsv.MarshalString(data); err == nil {
		// Prepend CSV comments bc cBioportal will result in errors without them
		if reflect.TypeOf(data) == reflect.TypeOf([]PatientData{}) {
			output = prefixDataClinicalPatient + output
		} else if reflect.TypeOf(data) == reflect.TypeOf([]SampleData{}) {
			output = prefixDataClinicalSample + output
		}

		if _, err := file.Write([]byte(output)); err != nil {
			return errors.New("file: In die Datei kann nicht geschrieben werden")
		}
	} else {
		return errors.New("file: Fehler beim Erstellen der Ausgabedaten")
	}

	return nil
}
