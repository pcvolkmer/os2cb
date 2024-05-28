package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type Patients struct {
	db *sql.DB
}

func InitPatients(db *sql.DB) Patients {
	return Patients{
		db: db,
	}
}

func (patients *Patients) FetchOcaPlusPatientIds() ([]string, error) {
	query := `SELECT DISTINCT patienten_id FROM dk_molekulargenetik 
		JOIN prozedur ON (prozedur.id = dk_molekulargenetik.id)
		JOIN patient ON (patient.id = prozedur.patient_id)
		WHERE panel = 'OCAPlus' AND procedur.geloescht = 0
		ORDER BY patienten_id;`

	var patientenIds []string

	if rows, err := db.Query(query); err == nil {
		var patientenId sql.NullString
		for rows.Next() {
			if err := rows.Scan(&patientenId); err == nil {
				patientenIds = append(patientenIds, patientenId.String)
			}
		}
	} else {
		return nil, err
	}

	return patientenIds, nil
}

func (patients *Patients) FetchBy(patientIDs []string, tkType string, allTk bool) ([]PatientData, error) {
	query := `SELECT DISTINCT
	   patient.patienten_id,
	   geschlecht,
	   DATE_FORMAT(FROM_DAYS(DATEDIFF(now(),geburtsdatum)), '%Y')+0 AS geburtsdatum,
	   sterbedatum,
	   ki.karnofsky
	   FROM patient
	   -- karnofsky
		  LEFT OUTER JOIN (
				SELECT patienten_id, karnofsky, MAX(p.beginndatum) FROM dk_ukw_tb_basisdaten dutb
					  JOIN dk_tumorkonferenz dt ON (dutb.id = dt.id AND dt.tk = ?)
					  JOIN prozedur p ON (p.id = dutb.id)
					  JOIN patient pat ON (pat.id = p.patient_id)
					  WHERE dutb.karnofsky IS NOT NULL AND p = 0
					  GROUP BY patienten_id
					  ORDER BY patienten_id
		  ) ki ON (ki.patienten_id = patient.patienten_id)
		  WHERE patient.patienten_id IN ('` + strings.Join(patientIDs, "','") + "') ORDER BY patient.patienten_id;"

	var results = []PatientData{}

	if rows, err := db.Query(query, tkType); err == nil {
		var patientenId sql.NullString
		var sex sql.NullString
		var geburtsdatum sql.NullInt16
		var sterbedatum sql.NullString
		var karnofsky sql.NullString

		for rows.Next() {
			if err := rows.Scan(&patientenId, &sex, &geburtsdatum, &sterbedatum, &karnofsky); err == nil {
				result := &PatientData{
					ID: AnonymizedID(patientenId.String),
				}

				// GENDER + SEX
				if sex, err := sex.Value(); err == nil && sex != nil {
					if sex == "m" {
						result.Sex = "Male"
					} else if sex == "w" {
						result.Sex = "Female"
					}

					if sex == "m" {
						result.Gender = "Male"
					} else if sex == "w" {
						result.Gender = "Female"
					}
					// Others - Code?
				}

				// AGE
				if geburtsdatum, err := geburtsdatum.Value(); err == nil && geburtsdatum != nil {
					result.Age = fmt.Sprint(geburtsdatum)
				}

				// OS_STATUS
				// OS_MONTHS applied using appendDiagnoseDaten()
				if sterbedatum, err := sterbedatum.Value(); err == nil && sterbedatum != nil {
					result.OsStatus = "DECEASED"
				} else {
					result.OsStatus = "LIVING"
				}

				if val, err := karnofsky.Value(); err == nil && val != nil {
					result.MtbEcogStatus = karnofskyToEcog(fmt.Sprintf("%s", val))
				} else {
					result.MtbEcogStatus = "NA"
				}

				result = appendDiagnoseDaten(patientenId.String, result, allTk)

				results = append(results, *result)
			}
		}

		return results, nil
	}

	return nil, fmt.Errorf("keine Daten zu Patienten gefunden")
}

// Ermittelt den ECOG anhand des Karnofsky-Grads
func karnofskyToEcog(karnofsky string) string {

	// Plain percent number
	karnofsky = strings.ReplaceAll(karnofsky, "%", "")
	karnofsky = strings.TrimSpace(karnofsky)

	if karnofskyGrade, err := strconv.Atoi(karnofsky); err == nil {
		if karnofskyGrade >= 90 {
			return "0"
		} else if karnofskyGrade >= 70 {
			return "1"
		} else if karnofskyGrade >= 50 {
			return "2"
		} else if karnofskyGrade >= 30 {
			return "3"
		} else if karnofskyGrade > 0 {
			return "4"
		} else {
			return "5"
		}
	}

	return "NA"
}

// Ermittelt Diagnosedaten zu den Patientendaten und gibt diese zurück
func appendDiagnoseDaten(patientID string, data *PatientData, allTk bool) *PatientData {
	query := `SELECT
    	icdo3histologie,
    	beginndatum,
    	icd10,
    	fernmetastasen,
    	pcve.shortdesc AS diagnose,
    	ROUND(DATEDIFF(IF(sterbedatum IS NULL, NOW(), sterbedatum),diagnosedatum) / 30) AS os_month
		FROM prozedur
		JOIN dk_diagnose ON prozedur.id = dk_diagnose.id
		JOIN property_catalogue_version_entry pcve ON pcve.code = icd10 AND pcve.property_version_id = icd10_propcat_version
		JOIN patient p on p.id = prozedur.patient_id
		JOIN erkrankung_prozedur ep ON ep.prozedur_id = prozedur.id
		WHERE prozedur.geloescht = 0 AND p.patienten_id = ? AND ep.erkrankung_id IN (
			SELECT ep.erkrankung_id FROM dk_tumorkonferenz
				JOIN prozedur pro on dk_tumorkonferenz.id = pro.id
				JOIN patient pat on pro.patient_id = pat.id
				JOIN erkrankung_prozedur ep ON ep.prozedur_id = pro.id
				WHERE pat.patienten_id = ? AND (dk_tumorkonferenz.tk = '27' OR 1 = ?)
				ORDER BY beginndatum DESC
		)
		ORDER BY beginndatum DESC;`

	var icdo3histologie sql.NullString
	var beginndatum sql.NullString
	var icd10 sql.NullString
	var fernmetastasen sql.NullString
	var diagnose sql.NullString
	var osMonth sql.NullInt16

	if row := db.QueryRow(query, patientID, patientID, allTk); row != nil {

		if err := row.Scan(&icdo3histologie, &beginndatum, &icd10, &fernmetastasen, &diagnose, &osMonth); err == nil {
			// OS_MONTH
			// Aktuell nur ganze Monate als Kommazahl (Anzahl Tage / 30)
			if osMonth, err := osMonth.Value(); err == nil && osMonth != nil {
				data.OsMonths = fmt.Sprintf("%d.0", osMonth)
			}

			// ICD-O3-Morphologie Code
			if icdo3histologie, err := icdo3histologie.Value(); err == nil && icdo3histologie != nil {
				data.IcdO3MorphCode = fmt.Sprint(icdo3histologie)
			}

			// ICD10 Code
			if icd10, err := icd10.Value(); err == nil && icd10 != nil {
				data.Icd10Code = fmt.Sprint(icd10)
			}

			// DIAGNOSIS
			if diagnose, err := diagnose.Value(); err == nil && diagnose != nil {
				data.Diagnosis = fmt.Sprint(diagnose)
			}

			// ONKOTREE_CODE
			data.OncotreeCode = "NA"

			// SPREAD_OF_DISEASE
			if fernmetastasen, err := fernmetastasen.Value(); err == nil && fernmetastasen != nil {
				if fernmetastasen == 1 {
					data.SpreadOfDisease = "metastasiert"
				}
			}
		}
	}

	// Erforderlich: Beruecksichtigung von Krankheiten in "Anamnesebogen"?
	queryErkrankungen := `SELECT DISTINCT pcve.shortdesc
			FROM prozedur
		    JOIN dk_diagnose ON prozedur.id = dk_diagnose.id
			JOIN property_catalogue_version_entry pcve ON pcve.code = icd10 AND pcve.property_version_id = icd10_propcat_version
			JOIN erkrankung_prozedur ep ON prozedur.id = ep.prozedur_id
			JOIN patient p on p.id = prozedur.patient_id
			WHERE p.patienten_id = ?
			ORDER BY beginndatum DESC`

	if rows, err := db.Query(queryErkrankungen, patientID); err == nil {

		var diags []string
		var erkrankung sql.NullString

		for rows.Next() {
			if err := rows.Scan(&erkrankung); err == nil {
				if erkrankung.String != data.Diagnosis {
					diags = append(diags, erkrankung.String)
				}
			}
		}

		data.PastMalignantDisease = strings.Join(diags, " + ")
	}

	return data
}

type PatientData struct {
	ID                       string `csv:"PATIENT_ID"`
	Gender                   string `csv:"GENDER"`
	Sex                      string `csv:"SEX"`
	Age                      string `csv:"AGE"`
	IcdO3MorphCode           string `csv:"ICD_O3_MORPH_CODE"`
	Diagnosis                string `csv:"DIAGNOSIS"`
	OncotreeCode             string `csv:"ONCOTREE_CODE"`
	Icd10Code                string `csv:"ICD_10_CODE"`
	SpreadOfDisease          string `csv:"SPREAD_OF_DISEASE"`
	MtbEcogStatus            string `csv:"MTB_ECOG_STATUS"`
	PastMalignantDisease     string `csv:"PAST_MALIGNANT_DISEASE"`
	PretherapyProgress       string `csv:"PREATHERAPY_PROGRESS"`
	NumSystemicPretherapy    string `csv:"NUM_SYSTEMIC_PRETHERAPY"`
	PretherapyMedication     string `csv:"PREATHERAPY_MEDICATION"`
	PretherapyMedicationNcit string `csv:"PREATHERAPY_MEDICATION_NCIT"`
	PretherapyBestResponse   string `csv:"PREATHERAPY_BEST_RESPONSE"`
	PretherapyPfs            string `csv:"PREATHERAPY_PFS"`
	OsStatus                 string `csv:"OS_STATUS"`
	OsMonths                 string `csv:"OS_MONTHS"`
}
