package main

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Samples struct {
	db          *sql.DB
	ocaPlusOnly bool
}

func InitSamples(db *sql.DB, ocaPlusOnly bool) Samples {
	return Samples{
		db:          db,
		ocaPlusOnly: ocaPlusOnly,
	}
}

// Aktuell alle Diagnosen/Erkrankungen des Patienten
func (samples *Samples) Fetch(patientID string) ([]SampleData, error) {
	checkQuery := `SELECT id FROM patient WHERE patienten_id = ?`
	if row := db.QueryRow(checkQuery, patientID); row != nil {
		var id string
		if err := row.Scan(&id); err != nil {
			return nil, fmt.Errorf("keine Daten zu Patient mit ID '%s'", patientID)
		}
	}

	query := `SELECT DISTINCT ep.erkrankung_id FROM prozedur pro
		JOIN patient pat on pro.patient_id = pat.id
		JOIN erkrankung_prozedur ep ON ep.prozedur_id = pro.id
		WHERE pat.patienten_id = ?
		ORDER BY beginndatum DESC`

	if rows, err := db.Query(query, patientID); err == nil {
		var erkrankungID string
		var result []SampleData

		for rows.Next() {
			if err := rows.Scan(&erkrankungID); err == nil {
				if samplesForDisease, err := fetchSamplesForDisease(patientID, erkrankungID, samples.ocaPlusOnly); err == nil {
					result = append(result, samplesForDisease...)
				}
			}
		}
		return result, nil
	}
	return nil, errors.New("fetch: No data found")
}

func fetchSamplesForDisease(patientID string, diseaseID string, ocaPlusOnly bool) ([]SampleData, error) {

	query := `SELECT
    	dm.id,
    	dm.datum,
    	dm.einsendenummer,
    	dm.probenmaterial,
    	dm.entnahmemethode,
    	dm.entnahmedatum,
    	dm.tumorzellgehalt,
    	dm.tumormutationalburden,
    	pcve.shortdesc as sample_location,
    	dm.nukleinsaeure,
    	pcve2.code as panel_code,
    	pcve2.shortdesc as panel
		FROM prozedur
		JOIN dk_molekulargenetik dm on prozedur.id = dm.id
		JOIN erkrankung_prozedur ep on prozedur.id = ep.prozedur_id
		LEFT JOIN property_catalogue_version_entry pcve ON pcve.code = icdo3lokalisation AND pcve.property_version_id = icdo3lokalisation_propcat_version
		LEFT JOIN property_catalogue_version_entry pcve2 ON pcve2.code = panel AND pcve2.property_version_id = panel_propcat_version
		WHERE prozedur.geloescht = 0 AND ep.erkrankung_id = ?
		ORDER BY beginndatum DESC`

	if rows, err := db.Query(query, diseaseID); err == nil {

		var result []SampleData

		anonymizedPatientID := AnonymizedID(patientID)

		var id sql.NullString
		var datum sql.NullString
		var einsendenummer sql.NullString
		var probenmaterial sql.NullString
		var entnahmemethode sql.NullString
		var entnahmedatum sql.NullString
		var tumorzellgehalt sql.NullString
		var tumormutationalburden sql.NullString
		var sampleLocation sql.NullString
		var nukleinsaeure sql.NullString
		var panelCode sql.NullString
		var panel sql.NullString

		for rows.Next() {
			data := SampleData{}

			if err := rows.Scan(
				&id,
				&datum,
				&einsendenummer,
				&probenmaterial,
				&entnahmemethode,
				&entnahmedatum,
				&tumorzellgehalt,
				&tumormutationalburden,
				&sampleLocation,
				&nukleinsaeure,
				&panelCode,
				&panel,
			); err == nil {

				// PANEL_CODE
				if panelCode, err := panelCode.Value(); err == nil && (panelCode != "OCAPlus" || !ocaPlusOnly) {
					continue
				}

				// SAMPLE_ID
				if einsendenummer, err := einsendenummer.Value(); err == nil && einsendenummer != nil {
					data.PatientID = anonymizedPatientID
					data.SampleID = AnonymizedID(sanitizeSampleId(fmt.Sprint(einsendenummer)))
				} else {
					continue
				}

				data.SampleLocRefPrimarus = "NA"

				// SAMPLE_LOC_REF_PRIMARIUS
				if value, err := probenmaterial.Value(); err == nil && value != nil {
					if value == "T" {
						data.SampleLocRefPrimarus = "Primaertumor"
					} else if value == "M" {
						data.SampleLocRefPrimarus = "Metastase"
					}
				}

				// SAMPLE_METHOD - nur bekannt "Biopsie" und "Resektat". Andere mögliche Werte?
				if value, err := entnahmemethode.Value(); err == nil && value != nil {
					if value == "B" {
						data.SampleMethod = "Biopsie"
					} else if value == "R" {
						data.SampleMethod = "Resektat"
					}
				}

				// SAMPLE_LOCATION
				if value, err := sampleLocation.Value(); err == nil && value != nil {
					data.SampleLocation = fmt.Sprint(value)
				} else {
					data.SampleLocation = "NA"
				}

				// SAMPLE_AGE
				if datum, err := datum.Value(); err == nil && datum != nil {
					if value, err := entnahmedatum.Value(); err == nil && value != nil {
						if datum, err := time.Parse("2006-01-02", fmt.Sprint(datum)); err == nil {
							if entnahmedatum, err := time.Parse("2006-01-02", fmt.Sprint(value)); err == nil {
								data.SampleAge = fmt.Sprintf("%f", datum.Sub(entnahmedatum).Hours()/24)
							}
						}
					}
				}

				// TUMOR_CELL_AMOUNT
				if value, err := tumorzellgehalt.Value(); err == nil && value != nil {
					data.TumorCellAmount = fmt.Sprint(value)
				} else {
					data.TumorCellAmount = "NA"
				}

				// SEQUENCING_DNA_PANEL / FUSION_RNA_PANEL
				// Initial values - wenn nicht anders angegeben
				data.SequencingDnaPanel = "NA"
				data.FusionRnaPanel = "NA"
				if value, err := nukleinsaeure.Value(); err == nil && value != nil {
					if panel, err := panel.Value(); err == nil && panel != nil {
						if value == "dna" {
							data.SequencingDnaPanel = fmt.Sprint(panel)
						} else if value == "rna" {
							data.FusionRnaPanel = fmt.Sprint(panel)
						} else if value == "dnarna" {
							data.SequencingDnaPanel = fmt.Sprint(panel)
							data.FusionRnaPanel = fmt.Sprint(panel)
						}
					}
				}

				// SEQUENCING_DNA_PLATFORM + SEQUENCING_RNA_PLATFORM
				// Initial values - wenn nicht anders angegeben
				data.SequencingDnaPlatform = "NA"
				data.SequencingRnaPlatform = "NA"
				if value, err := nukleinsaeure.Value(); err == nil && value != nil {
					if panelCode, err := panelCode.Value(); err == nil && panelCode != nil {
						// Keine Onkostar-Dokumentation in Wuerzburg
						// Implementierung fuer WUE_
						if strings.HasPrefix(data.SampleID, "WUE_") {
							// DNA
							if (value == "dna" || value == "dnarna") && (panelCode == "OCAPlus" || panelCode == "OncomineV3" || panelCode == "OFA") {
								data.SequencingDnaPlatform = "Thermo Fisher"
							}
							// RNA
							if (value == "rna" || value == "dnarna") && (panelCode == "OCAPlus" || panelCode == "AFPLung" || panelCode == "AFPSarc") {
								data.SequencingRnaPlatform = "Thermo Fisher"
							}
						}
					}
				}

				// TMB_SCORE aus alten Formular "OS.Molekulargenetik" vor rev 81
				if value, err := tumormutationalburden.Value(); err == nil && value != nil {
					data.TmbScore = fmt.Sprint(value)
				} else {
					data.TmbScore = "NA"
				}

				if id, err := id.Value(); err == nil && id != nil {
					data = *immunhisto(fmt.Sprint(id), &data)
					data = *msi(fmt.Sprint(id), &data)

					// TMB_SCORE aus neuem Formular "OS.Molekulargenetik" ab rev 81
					data = *tmb(fmt.Sprint(id), &data)

					// GIM_/HRD_SCORE
					data.GimScore = "NA"
					data.HrdScore = "NA"
					if panelCode, err := panelCode.Value(); err == nil && panelCode == "OCAPlus" {
						data.GimScore, _ = hrd(fmt.Sprint(id))
					} else {
						data.HrdScore, _ = hrd(fmt.Sprint(id))
					}

					// HRD-LOH, TAI, LST
					// NA for now
					data.HrdLoh = "NA"
					data.Tai = "NA"
					data.Lst = "NA"
				}

				data.Her2Fish = "NA"
				data.OtherExamination = "NA"
				data.OtherIhc = "NA"
				data.DakoScore = "NA"
				data.Fusions = "NA"
				data.SpliceVariants = "NA"
				data.Mutations = "NA"
				data.Cnv = "NA"

			}

			result = append(result, data)
		}

		return result, nil
	}

	return nil, errors.New("Kann Daten nicht abrufen")
}

// Schreibt die Werte TPS, ICS und CPS in bestehende Probendaten und gibt diese dann wieder zurück.
func immunhisto(prozedurID string, sampleData *SampleData) *SampleData {
	query := `SELECT gen, tps, ic_score, cps FROM dk_molekularimmunhisto
    	JOIN prozedur_prozedur pp ON pp.prozedur2 = dk_molekularimmunhisto.id
    	WHERE pp.prozedur1 = ?`

	var gen sql.NullString
	var tps sql.NullString
	var icScore sql.NullString
	var cps sql.NullString

	// Initial values
	sampleData.Tps = "NA"
	sampleData.Ics = "NA"
	sampleData.Cps = "NA"

	if row := db.QueryRow(query, prozedurID); row != nil {
		if err := row.Scan(&gen, &tps, &icScore, &cps); err == nil {
			if value, err := gen.Value(); err == nil && value != nil {
				if value == "PDL1" {
					// TPS
					if value, err := tps.Value(); err == nil && value != nil {
						sampleData.Tps = fmt.Sprint(value)
					}
					// ICS
					if value, err := icScore.Value(); err == nil && value != nil {
						sampleData.Ics = fmt.Sprint(value)
					}
					// CPS
					if value, err := cps.Value(); err == nil && value != nil {
						sampleData.Cps = fmt.Sprint(value)
					}
				}
			}
		}
	}

	return sampleData
}

// Schreibt die Werte MSI in bestehende Probendaten und gibt diese dann wieder zurück.
func msi(prozedurID string, sampleData *SampleData) *SampleData {
	query := `SELECT seqprozentwert FROM prozedur_prozedur pp
		JOIN dk_molekulargenetik ON pp.prozedur1 = dk_molekulargenetik.id
		JOIN dk_molekluargenmsi ON pp.prozedur2 = dk_molekluargenmsi.id
		WHERE pp.prozedur1 = ? AND komplexerbiomarker = 'MSI'`

	var prozentwert sql.NullString

	// Initial values
	sampleData.MsiPanel = "NA"
	sampleData.MsiPcr = "NA"
	sampleData.MsiIg = "NA"

	if rows, err := db.Query(query, prozedurID); err == nil {
		for rows.Next() {
			if err := rows.Scan(&prozentwert); err == nil {
				if prozentwert, err := prozentwert.Value(); err == nil && prozentwert != nil {
					sampleData.MsiPanel = fmt.Sprint(prozentwert)
				}
			}
		}
	}

	return sampleData
}

// Ermittelt den HRD Score für angegebene Hauptprozedur
func hrd(prozedurID string) (string, error) {
	query := `SELECT score FROM prozedur_prozedur pp
		JOIN dk_molekulargenetik ON pp.prozedur1 = dk_molekulargenetik.id
		JOIN dk_molekluargenmsi ON pp.prozedur2 = dk_molekluargenmsi.id
		WHERE pp.prozedur1 = ? AND komplexerbiomarker = 'HRD'`

	var score sql.NullString

	if rows, err := db.Query(query, prozedurID); err == nil {
		for rows.Next() {
			if err := rows.Scan(&score); err == nil {
				if score.Valid {
					return score.String, nil
				}
			}
		}
	}

	return "NA", fmt.Errorf("No HRD Score entry found")
}

func tmb(prozedurID string, sampleData *SampleData) *SampleData {
	query := `SELECT tumormutationalburden FROM dk_molekluargenmsi
    	JOIN prozedur_prozedur pp ON pp.prozedur2 = dk_molekluargenmsi.id
		WHERE tumormutationalburden IS NOT NULL AND pp.prozedur1 = ?`

	var tumormutationalburden sql.NullString

	if rows, err := db.Query(query, prozedurID); err == nil {
		for rows.Next() {
			if err := rows.Scan(&tumormutationalburden); err == nil {
				if tumormutationalburden, err := tumormutationalburden.Value(); err == nil && tumormutationalburden != nil {
					sampleData.TmbScore = fmt.Sprint(tumormutationalburden)
				}
			}
		}
	}

	return sampleData
}

func sanitizeSampleId(id string) string {
	re := regexp.MustCompile("(?P<Letter>[A-Z])/[0-9]{2}(?P<Year2>[0-9]{2})/(?P<LfdNr>[0-9]+)")
	if re.MatchString(id) {
		matches := re.FindStringSubmatch(id)
		return fmt.Sprintf(
			"%s%s-%s",
			matches[re.SubexpIndex("Letter")],
			matches[re.SubexpIndex("LfdNr")],
			matches[re.SubexpIndex("Year2")],
		)
	}
	return id
}

type SampleData struct {
	PatientID             string `csv:"PATIENT_ID"`
	SampleID              string `csv:"SAMPLE_ID"`
	SampleLocRefPrimarus  string `csv:"SAMPLE_LOC_REF_PRIMARUS"`
	SampleMethod          string `csv:"SAMPLE_METHOD"`
	SampleLocation        string `csv:"SAMPLE_LOCATION"`
	SampleAge             string `csv:"SAMPLE_AGE"`
	TumorCellAmount       string `csv:"TUMOR_CELL_AMOUNT"`
	SequencingDnaPanel    string `csv:"SEQUENCING_DNA_PANEL"`
	SequencingDnaPlatform string `csv:"SEQUENCING_DNA_PLATFORM"`
	FusionRnaPanel        string `csv:"FUSION_RNA_PANEL"`
	SequencingRnaPlatform string `csv:"SEQUENCING_RNA_PLATFORM"`
	TmbScore              string `csv:"TMB_SCORE"`
	Tps                   string `csv:"TPS"`
	Ics                   string `csv:"ICS"`
	Cps                   string `csv:"CPS"`
	MsiIg                 string `csv:"MSI_IG"`
	MsiPcr                string `csv:"MSI_PCR"`
	MsiPanel              string `csv:"MSI_PANEL"`
	Her2Fish              string `csv:"HER2_FISH"`
	OtherExamination      string `csv:"OTHER_EXAMINATION"`
	OtherIhc              string `csv:"OTHER_IHC"`
	DakoScore             string `csv:"DAKO_SCORE"`
	Fusions               string `csv:"FUSIONS"`
	SpliceVariants        string `csv:"SPLICE_VARIANTS"`
	Mutations             string `csv:"MUTATIONS"`
	Cnv                   string `csv:"CNV"`
	GimScore              string `csv:"GIM_SCORE"`
	HrdScore              string `csv:"HRD_SCORE"`
	HrdLoh                string `csv:"HRD_LOH"`
	Tai                   string `csv:"TAI"`
	Lst                   string `csv:"LST"`
}

func SampleDataHeaders() []string {
	return []string{
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
		"HRD_LOH",
		"TAI",
		"LST",
	}
}

func (data *SampleData) AsStringArray() []string {
	return []string{
		data.PatientID,
		data.SampleID,
		data.SampleLocRefPrimarus,
		data.SampleMethod,
		data.SampleLocation,
		data.SampleAge,
		data.TumorCellAmount,
		data.SequencingDnaPanel,
		data.SequencingDnaPlatform,
		data.FusionRnaPanel,
		data.SequencingRnaPlatform,
		data.TmbScore,
		data.Tps,
		data.Ics,
		data.Cps,
		data.MsiIg,
		data.MsiPcr,
		data.MsiPanel,
		data.Her2Fish,
		data.OtherExamination,
		data.OtherIhc,
		data.DakoScore,
		data.Fusions,
		data.SpliceVariants,
		data.Mutations,
		data.Cnv,
		data.GimScore,
		data.HrdScore,
	}
}
