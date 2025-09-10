package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/gocarina/gocsv"
	"github.com/xuri/excelize/v2"
)

//go:embed resources/prefix-data_clinical_patient.tsv
var prefixDataClinicalPatient string

//go:embed resources/prefix-data_clinical_sample.tsv
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

// Schreibt Daten in Xlsx Datei
func WriteXlsxFile(filename string, patientData []PatientData, sampleData []SampleData) error {
	file := excelize.NewFile()
	defer func() {
		if err := file.Close(); err != nil {
			return
		}
	}()

	patientsIndex, _ := file.NewSheet("Patients Data")
	samplesIndex, _ := file.NewSheet("Samples Data")

	_ = addPatientData(file, patientsIndex, patientData)
	_ = addSampleData(file, samplesIndex, sampleData)

	_ = file.DeleteSheet("Sheet1")

	if err := file.SaveAs(filename); err != nil {
		fmt.Println(err)
	}

	return nil
}

func addPatientData(file *excelize.File, index int, patientData []PatientData) error {
	file.SetActiveSheet(index)

	for idx, columnHeader := range PatientDataHeaders() {
		cell := getExcelColumn(idx) + "1"
		_ = file.SetCellValue("Patients Data", cell, columnHeader)
	}

	for row, data := range patientData {
		for idx, value := range data.AsStringArray() {
			cell := getExcelColumn(idx) + fmt.Sprint(row+2)
			_ = file.SetCellValue("Patients Data", cell, value)
		}
	}

	return nil
}

func addSampleData(file *excelize.File, index int, sampleData []SampleData) error {
	file.SetActiveSheet(index)

	for idx, columnHeader := range SampleDataHeaders() {
		cell := getExcelColumn(idx) + "1"
		_ = file.SetCellValue("Samples Data", cell, columnHeader)
	}

	for row, data := range sampleData {
		for idx, value := range data.AsStringArray() {
			cell := getExcelColumn(idx) + fmt.Sprint(row+2)
			_ = file.SetCellValue("Samples Data", cell, value)
		}
	}

	return nil
}

func getExcelColumn(idx int) string {
	z := int('Z' - 'A' + 1)
	m := idx % z
	if idx <= m {
		return string(rune(idx + 'A'))
	}

	r := ((idx - m) / z) - 1
	return string(rune(r+'A')) + string(rune(m+'A'))
}
