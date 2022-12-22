# Onkostar zu cBioportal Bayern

Ziel dieser Anwendung ist der Export relevanter anonymisierter Patientendaten und Daten zu Tumorproben aus Onkostar,
damit diese in eine bayernweite cBioportal Installation importiert werden können.

## Anwendung

Der Parameter `--help` zeigt folgenden Hilfetext an

```
Usage: os2cb <command>

A simple tool to export data from Onkostar into TSV file format for cBioportal

Flags:
  -h, --help                         Show context-sensitive help.
  -U, --user=STRING                  Database username
  -P, --password=STRING              Database password
  -H, --host="localhost"             Database host
      --port=3306                    Database port
  -D, --database="onkostar"          Database name
      --patient-id=PATIENT-ID,...    PatientenIDs der zu exportierenden Patienten. Kommagetrennt bei mehreren IDs
      --filename=STRING              Exportiere in diese Datei
      --append                       An bestehende Datei anhängen
      --csv                          Verwende CSV-Format anstelle TSV-Format. Trennung mit ';' für MS Excel

Commands:
  export-patients    Export patient data
  export-samples     Export sample data
```
