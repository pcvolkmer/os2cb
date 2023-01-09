package main

import (
	"database/sql"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"strings"
)

func ShowBrowser(patientIds []string, db *sql.DB) {
	browser := Browser{
		patientIds: patientIds,
		db:         db,
	}
	browser.show()
}

type Browser struct {
	db                *sql.DB
	patientIds        []string
	currentPatientIds []string
}

func (browser *Browser) show() {
	grid := tview.NewGrid()
	grid.SetRows(3, 0)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	item := tview.NewInputField().SetLabel("Patienten-IDs (kommagetrennt)").SetText(strings.Join(cli.PatientId, ","))
	item.SetChangedFunc(func(text string) {
		ids := strings.Split(text, ",")
		patientIds := []string{}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			patientIds = append(patientIds, id)
			browser.currentPatientIds = patientIds
		}
	})
	item.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyCR {
			if flex.GetItemCount() > 0 {
				flex.RemoveItem(flex.GetItem(0))
			}
			if flex.GetItemCount() > 0 {
				flex.RemoveItem(flex.GetItem(0))
			}
			flex = browser.addPatientTable(flex, browser.currentPatientIds)
			flex = browser.addSampleTable(flex, browser.currentPatientIds)
		}
	})
	grid.AddItem(item, 0, 0, 1, 1, 0, 0, true)

	flex = browser.addPatientTable(flex, cli.PatientId)
	flex = browser.addSampleTable(flex, cli.PatientId)

	grid.AddItem(flex, 1, 0, 1, 1, 0, 0, false)

	app := tview.NewApplication()

	if err := app.SetRoot(grid, true).Run(); err == nil {
		os.Exit(0)
	}

}

func (browser *Browser) addPatientTable(flex *tview.Flex, patientIds []string) *tview.Flex {
	if data, err := FetchAllPatientData(patientIds, browser.db); err == nil {
		table := tview.NewTable()
		table.SetBorder(true)
		table.SetBorders(true)
		table.SetTitle("patient")

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
		}

		for idx, item := range headline {
			table.SetCellSimple(0, idx, item)
		}

		for idx, item := range data {
			table.SetCellSimple(idx+1, 0, item.Id)
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
		}

		flex.AddItem(table, 0, 1, false)
	}

	return flex
}

func (browser *Browser) addSampleTable(flex *tview.Flex, patientIds []string) *tview.Flex {
	if data, err := FetchAllSampleData(patientIds, browser.db); err == nil {
		table := tview.NewTable()
		table.SetBorder(true)
		table.SetBorders(true)
		table.SetTitle("sample")

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
		}

		for idx, item := range headline {
			table.SetCellSimple(0, idx, item)
		}

		for idx, item := range data {
			table.SetCellSimple(idx+1, 0, item.PatientId)
			table.SetCellSimple(idx+1, 1, item.SampleId)
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
		}

		flex.AddItem(table, 0, 1, false)
	}

	return flex
}
