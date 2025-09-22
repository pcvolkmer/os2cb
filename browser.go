package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func contains(patientIds []string, patientID string) bool {
	for _, elem := range patientIds {
		if elem == patientID {
			return true
		}
	}
	return false
}

type BrowserType int8

const (
	Patient BrowserType = iota
	Sample              = iota
)

type Browser struct {
	db             *sql.DB
	browserType    BrowserType
	patientIds     []string
	checkSampleIds bool

	app        *tview.Application
	grid       *tview.Grid
	inputField *tview.InputField
	dropDown   *tview.DropDown
	table      *tview.Table
}

func NewBrowser(patientIds []string, checkSampleIds bool, db *sql.DB) *Browser {
	var inputField *tview.InputField
	var dropDown *tview.DropDown

	browser := &Browser{
		db:             db,
		browserType:    Patient,
		patientIds:     patientIds,
		checkSampleIds: checkSampleIds,
		app:            tview.NewApplication(),
		grid:           tview.NewGrid().SetRows(2, 2, 0, 1),
	}

	inputField = tview.NewInputField().
		SetLabel("Patienten-IDs (kommagetrennt): ").
		SetLabelWidth(32).SetText(strings.Join(cli.PatientID, ","))
	inputField.SetChangedFunc(func(text string) {
		ids := strings.Split(text, ",")
		var patientIds []string
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if !contains(patientIds, id) {
				patientIds = append(patientIds, id)
				browser.patientIds = patientIds
			}
		}
	}).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyCR {
			browser.replaceTable()
			browser.app.SetFocus(browser.dropDown)
		}
	})
	inputField.SetBorderPadding(0, 1, 1, 1)
	browser.grid.AddItem(inputField, 0, 0, 1, 1, 0, 0, true)

	dropDown = tview.NewDropDown().SetLabel("Ansicht: ").SetLabelWidth(32)
	dropDown.SetOptions([]string{"Patienten-Daten", "Sample-Daten"}, func(text string, index int) {
		if text == "Patienten-Daten" {
			browser.browserType = Patient
		} else if text == "Sample-Daten" {
			browser.browserType = Sample
		} else {
			return
		}
		browser.replaceTable()
		browser.app.SetFocus(browser.table)
	})
	if browser.browserType == Sample {
		dropDown.SetCurrentOption(1)
	} else {
		dropDown.SetCurrentOption(0)
	}

	dropDown.SetBorderPadding(0, 1, 1, 1)
	browser.grid.AddItem(dropDown, 1, 0, 1, 1, 0, 0, false)

	browser.inputField = inputField
	browser.dropDown = dropDown

	browser.replaceTable()

	info := tview.NewFlex()
	info.AddItem(tview.NewTextView().SetLabel("<Strg>+C ").SetText("Beenden").SetTextColor(tcell.ColorGray), 20, 0, false)
	info.AddItem(tview.NewTextView().SetLabel("<Strg>+S ").SetText("Tabelle speichern").SetTextColor(tcell.ColorGray), 30, 0, false)
	info.AddItem(tview.NewTextView(), 0, 1, false)
	browser.grid.AddItem(info, 3, 0, 1, 1, 0, 0, false)

	browser.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if browser.inputField.HasFocus() {
				browser.app.SetFocus(browser.dropDown)
			} else if browser.dropDown.HasFocus() {
				browser.app.SetFocus(browser.table)
			} else if browser.table.HasFocus() {
				browser.app.SetFocus(browser.inputField)
			}
		} else if event.Key() == tcell.KeyBacktab {
			if browser.inputField.HasFocus() {
				browser.app.SetFocus(browser.table)
			} else if browser.dropDown.HasFocus() {
				browser.app.SetFocus(browser.inputField)
			} else if browser.table.HasFocus() {
				browser.app.SetFocus(browser.dropDown)
			}
		} else if event.Key() == tcell.KeyCtrlS {
			focus := browser.app.GetFocus()
			var filename string
			filenameInput := tview.NewInputField().SetLabel("Datei: ")
			filenameInput.SetChangedFunc(func(text string) {
				filename = text
			}).SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyCR {
					if err := browser.saveTable(filename); err == nil {
						browser.app.SetFocus(focus)
						info.RemoveItem(filenameInput)
					}
				} else if key == tcell.KeyEsc {
					browser.app.SetFocus(focus)
					info.RemoveItem(filenameInput)
				}
			})
			info.AddItem(filenameInput, 0, 1, true)
			browser.app.SetFocus(filenameInput)
		}
		return event
	})

	return browser
}

func (browser *Browser) Show() {
	if err := browser.app.SetRoot(browser.grid, true).Run(); err == nil {
		err := db.Close()
		if err != nil {
			log.Println("Cannot close database connection")
		}
		os.Exit(0)
	}
}

func (browser *Browser) saveTable(filename string) error {
	if browser.browserType == Patient {
		if data, err := FetchAllPatientData(browser.patientIds, browser.db); err == nil {
			return WriteFile(filename, data)
		} else {
			return err
		}
	} else if browser.browserType == Sample {
		if data, err := FetchAllSampleData(browser.patientIds, browser.db); err == nil {
			return WriteFile(filename, data)
		} else {
			return err
		}
	}
	return nil
}

func (browser *Browser) replaceTable() {
	if browser.table != nil {
		browser.grid.RemoveItem(browser.table)
	}

	if t, err := browser.createTable(browser.patientIds); err == nil {
		browser.table = t
		browser.grid.AddItem(t, 2, 0, 1, 1, 0, 0, false)
	}
}

func (browser *Browser) createTable(patientIds []string) (*tview.Table, error) {
	var table *tview.Table

	if browser.browserType == Sample {
		if t, err := browser.createSampleTable(patientIds); err != nil {
			table = t
		} else {
			return t, err
		}
	} else {
		if t, err := browser.createPatientTable(patientIds); err != nil {
			table = t
		} else {
			return t, err
		}
	}

	return table, nil
}

func (browser *Browser) createPatientTable(patientIds []string) (*tview.Table, error) {
	if data, err := FetchAllPatientData(patientIds, browser.db); err == nil {
		table := tview.NewTable()
		table.SetBorder(true)
		table.SetBorders(true)
		table.SetTitle(fmt.Sprintf("Patienten-Daten - %d Einträge", len(data)))

		headline := []string{
			"PATIENT_ID",
			"GENDER",
			"SEX",
			"AGE",
			"ICD_O3_MORPH_CODE",
			"DIAGNOSIS",
			"ONCOTREE_CODE",
			"ICD_10_CODE",
			"SPREAD_OF_DISEASE",
			"MTB_ECOG_STATUS",
			"PAST_MALIGNANT_DISEASE",
			"PREATHERAPY_PROGRESS",
			"NUM_SYSTEMIC_PRETHERAPY",
			"PREATHERAPY_MEDICATION",
			"PREATHERAPY_MEDICATION_NCIT",
			"PREATHERAPY_BEST_RESPONSE",
			"PREATHERAPY_PFS",
			"OS_STATUS",
			"OS_MONTHS",
			"DFS_STATUS",
			"DFS_MONTHS",
		}

		for idx, item := range headline {
			table.SetCellSimple(0, idx, item)
		}

		for idx, item := range data {
			table.SetCellSimple(idx+1, 0, item.ID)
			table.SetCellSimple(idx+1, 1, item.Gender)
			table.SetCellSimple(idx+1, 2, item.Sex)
			table.SetCellSimple(idx+1, 3, item.Age)
			table.SetCellSimple(idx+1, 4, item.IcdO3MorphCode)
			table.SetCellSimple(idx+1, 5, item.Diagnosis)
			table.SetCellSimple(idx+1, 6, item.OncotreeCode)
			table.SetCellSimple(idx+1, 7, item.Icd10Code)
			table.SetCellSimple(idx+1, 8, item.SpreadOfDisease)
			table.SetCellSimple(idx+1, 9, item.MtbEcogStatus)
			table.SetCellSimple(idx+1, 10, item.PastMalignantDisease)
			table.SetCellSimple(idx+1, 11, item.PretherapyProgress)
			table.SetCellSimple(idx+1, 12, item.NumSystemicPretherapy)
			table.SetCellSimple(idx+1, 13, item.PretherapyMedication)
			table.SetCellSimple(idx+1, 14, item.PretherapyMedicationNcit)
			table.SetCellSimple(idx+1, 15, item.PretherapyBestResponse)
			table.SetCellSimple(idx+1, 16, item.PretherapyPfs)
			table.SetCellSimple(idx+1, 17, item.OsStatus)
			table.SetCellSimple(idx+1, 18, item.OsMonths)
			table.SetCellSimple(idx+1, 19, item.DfsStatus)
			table.SetCellSimple(idx+1, 20, item.DfsMonths)
		}

		return table, nil
	} else {
		return nil, err
	}
}

func (browser *Browser) createSampleTable(patientIds []string) (*tview.Table, error) {
	if data, err := FetchAllSampleData(patientIds, browser.db); err == nil {
		table := tview.NewTable()
		table.SetBorder(true)
		table.SetBorders(true)
		table.SetTitle(fmt.Sprintf("Sample-Daten - %d Einträge", len(data)))

		headline := []string{
			"PATIENT_ID",
			"SAMPLE_ID",
			"SAMPLE_LOC_REF_PRIMARUS",
			"SAMPLE_METHOD",
			"SAMPLE_LOCATION",
			"SAMPLE_AGE",
			"TUMOR_CELL_AMOUNT",
			"SEQUENCING_DNA_PANEL",
			"SEQUENCING_DNA_PLATFORM",
			"FUSION_RNA_PANEL",
			"SEQUENCING_RNA_PLATFORM",
			"TMB_SCORE",
			"TPS",
			"ICS",
			"CPS",
			"MSI_IG",
			"MSI_PCR",
			"MSI_PANEL",
			"HER2_FISH",
			"OTHER_EXAMINATION",
			"OTHER_IHC",
			"DAKO_SCORE",
			"FUSIONS",
			"SPLICE_VARIANTS",
			"MUTATIONS",
			"CNV",
			"GIM_SCORE",
			"HRD_SCORE",
			"LST",
			"TAI",
			"HRD_LOH",
		}

		for idx, item := range headline {
			table.SetCellSimple(0, idx, item)
		}

		for idx, item := range data {
			sampleRegExp, err := regexp.Compile("^[A-Z]\\d+-\\d{2}$")

			table.SetCellSimple(idx+1, 0, item.PatientID)
			if err == nil && browser.checkSampleIds {
				if !sampleRegExp.MatchString(item.SampleID) {
					tableCell := tview.NewTableCell(item.SampleID).SetTextColor(tcell.ColorRed)
					table.SetCell(idx+1, 1, tableCell)
				} else {
					tableCell := tview.NewTableCell(item.SampleID).SetTextColor(tcell.ColorGreen)
					table.SetCell(idx+1, 1, tableCell)
				}
			} else {
				table.SetCellSimple(idx+1, 1, item.SampleID)
			}
			table.SetCellSimple(idx+1, 2, item.SampleLocRefPrimarus)
			table.SetCellSimple(idx+1, 3, item.SampleMethod)
			table.SetCellSimple(idx+1, 4, item.SampleLocation)
			table.SetCellSimple(idx+1, 5, item.SampleAge)
			table.SetCellSimple(idx+1, 6, item.TumorCellAmount)
			table.SetCellSimple(idx+1, 7, item.SequencingDnaPanel)
			table.SetCellSimple(idx+1, 8, item.SequencingDnaPlatform)
			table.SetCellSimple(idx+1, 9, item.FusionRnaPanel)
			table.SetCellSimple(idx+1, 10, item.SequencingRnaPlatform)
			table.SetCellSimple(idx+1, 11, item.TmbScore)
			table.SetCellSimple(idx+1, 12, item.Tps)
			table.SetCellSimple(idx+1, 13, item.Ics)
			table.SetCellSimple(idx+1, 14, item.Cps)
			table.SetCellSimple(idx+1, 15, item.MsiIg)
			table.SetCellSimple(idx+1, 16, item.MsiPcr)
			table.SetCellSimple(idx+1, 17, item.MsiPanel)
			table.SetCellSimple(idx+1, 18, item.Her2Fish)
			table.SetCellSimple(idx+1, 19, item.OtherExamination)
			table.SetCellSimple(idx+1, 20, item.OtherIhc)
			table.SetCellSimple(idx+1, 21, item.DakoScore)
			table.SetCellSimple(idx+1, 22, item.Fusions)
			table.SetCellSimple(idx+1, 23, item.SpliceVariants)
			table.SetCellSimple(idx+1, 24, item.Mutations)
			table.SetCellSimple(idx+1, 25, item.Cnv)
			table.SetCellSimple(idx+1, 26, item.GimScore)
			table.SetCellSimple(idx+1, 27, item.HrdScore)
			table.SetCellSimple(idx+1, 28, item.Lst)
			table.SetCellSimple(idx+1, 29, item.Tai)
			table.SetCellSimple(idx+1, 30, item.HrdLoh)
		}

		return table, nil
	} else {
		return nil, err
	}
}
