<p align="center" style="margin-bottom: 0px !important;">
  <img src="./assets/sync_512.png" width="120" align="center">
</p>
<h1 align="center">Nextclone</h1>
<p align="center" style="margin-bottom: 0px !important;">
  Eine einfache Desktop-App zum Sichern lokaler Ordner in Nextcloud.
</p>

<p align="center">
  <a href="README.md">English</a> | Deutsch
</p>

## Was Nextclone macht

- Kopiert oder synchronisiert Ordner von deinem Computer nach Nextcloud.
- Ermöglicht mehrere Backup-Jobs.
- Kann Jobs manuell oder nach Zeitplan ausführen.
- Kann beim Anmelden im Hintergrund starten.
- Zeigt Protokolle für jeden Job an.
- Enthält rclone in den Release-Downloads, sodass du es normalerweise nicht separat installieren musst.

## Installation

Du musst keinen Quellcode herunterladen und keine GitHub-Werkzeuge verwenden. Nimm einfach den vorbereiteten Download für dein Betriebssystem.

1. Öffne das [neueste Nextclone-Release](https://github.com/marvinscham/nextclone/releases/latest).
2. Scrolle zum Bereich "Assets".
3. Lade die Datei für dein Betriebssystem herunter.

### Windows

1. Lade `nextclone_v..._windows_amd64.zip` herunter.
2. Öffne die heruntergeladene Zip-Datei.
3. Entpacke sie in einen Ordner, zum Beispiel auf den Desktop oder in den Dokumente-Ordner.
4. Starte Nextclone mit einem Doppelklick auf `nextclone-windows-amd64.exe`.

Wenn Windows eine Warnung anzeigt, weil die App neu oder nicht signiert ist, wähle "Weitere Informationen" und dann "Trotzdem ausführen". Mache das nur, wenn du die Datei von der offiziellen Nextclone-Release-Seite heruntergeladen hast.

### Linux

Für Debian, Ubuntu, Linux Mint und ähnliche Distributionen:

1. Lade `nextclone_v..._linux_amd64.deb` herunter.
2. Öffne die heruntergeladene Datei mit deiner Software-Installation.
3. Klicke auf Installieren.
4. Starte Nextclone über den App-Starter.

Wenn deine Linux-Distribution keine `.deb`-Dateien unterstützt, lade `nextclone_v..._linux_amd64.zip` herunter, entpacke die Datei und starte `nextclone-linux-amd64`.

## Ersteinrichtung

Verbinde Nextclone zuerst mit deinem Nextcloud-Konto, bevor du einen Sync-Job erstellst.

1. Öffne Nextclone.
2. Klicke auf "Remote einrichten".
3. Gib die Adresse deines Nextcloud-Servers ein, zum Beispiel `https://cloud.example.com`.
4. Gib deinen Nextcloud-Benutzernamen ein.
5. Klicke auf "App-Passwort erstellen", wenn du ein Nextcloud-App-Passwort brauchst.
6. Füge das App-Passwort in Nextclone ein.
7. Klicke auf "Remote erstellen".

Ein App-Passwort ist sicherer als dein normales Nextcloud-Passwort. Du kannst es später bei Bedarf in den Sicherheitseinstellungen von Nextcloud entfernen.

## Backup-Job erstellen

1. Klicke auf "Sync hinzufügen".
2. Wähle einen Namen für den Job, zum Beispiel "Dokumente-Backup".
3. Wähle den lokalen Ordner aus, den du hochladen möchtest.
4. Gib den Remote-Namen ein, den du bei der Einrichtung erstellt hast, normalerweise `nextcloud`.
5. Gib den Zielordner in Nextcloud ein, zum Beispiel `/Backups/Dokumente`.
6. Wähle den Modus.
7. Speichere den Job.
8. Klicke auf "Starten", um ihn auszuführen.

### Kopieren oder Synchronisieren

- `copy` lädt neue und geänderte Dateien hoch, löscht aber keine Dateien aus Nextcloud. Das ist für die meisten Personen die sicherere Wahl.
- `sync` macht den Nextcloud-Ordner genauso wie deinen lokalen Ordner. Lokal gelöschte Dateien können dadurch auch aus Nextcloud gelöscht werden.

Verwende `copy`, außer du möchtest ausdrücklich eine exakte Spiegelung.

## Automatische Backups

Aktiviere beim Erstellen oder Bearbeiten eines Jobs "Automatisch ausführen" und wähle aus, wie oft der Job laufen soll.

Damit geplante Backups nach dem Anmelden laufen können, öffne "Einstellungen" und aktiviere "Nextclone beim Anmelden im Hintergrund starten".

## Updates

Nextclone prüft beim Start, ob eine neue Version verfügbar ist. Wenn ein Update verfügbar ist, wird der Update-Button im oberen Menü grün. Klicke darauf, um die neue Version zu installieren.

## Entwicklung

Informationen zu Entwicklung, Build, Zeitplanung, rclone und Releases stehen in [DEVELOPMENT.md](DEVELOPMENT.md).

## Icon-Credits

[Sync icons created by Freepik - Flaticon](https://www.flaticon.com/free-icons/sync)
