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
      --csv                          Verwende CSV-Format anstelle TSV-Format (UTF-16 und Trennung mit ';' zur Verwendung mit MS Excel)

Commands:
  export-patients    Export patient data
  export-samples     Export sample data
```

### Hinweis zu Passwörtern

Wird das Passwort nicht als Parameter angegeben, so wird im Anschluss danach gefragt.

### Hinweis zum Export im CSV-Format

Für den Export im CSV-Format wird zur Kompatibilität mit MS Excel der UTF-16 Zeichensatz verwendet. Das Trennzeichen ist dabei `;`.
Es handelt sich daher **nicht** um eine CSV-Datei nach [RFC 4180](https://www.rfc-editor.org/rfc/rfc4180).

### Anonymisierung

Die angegebenen IDs der Patienten als auch ermittelten IDs der Proben werden anonymisiert.
Dazu werden aus einer ID ein SHA256-Hash gebildet und von diesem die ersten 10 Zeichen zuzüglich Prefix `WUE_` für den
Export verwendet.

## Bekannte Probleme

Der Export als CSV-Datei erfolgt derzeit noch unter Verwendung von UTF-8, was zu Problemen bei der Darstellung von
Umlauten beim Öffnen mit MS Excel führt.
