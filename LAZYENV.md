# lazyenv

> TUI interattiva per gestire file `.env` — scritta in Go con Bubble Tea.

---

## Panoramica

lazyenv e' un tool da terminale che semplifica la gestione dei file `.env` nei progetti software. Permette di visualizzare, confrontare, editare e validare variabili d'ambiente tramite un'interfaccia TUI navigabile con tastiera.

---

## Stack tecnico

- **Linguaggio:** Go 1.22+
- **TUI framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Componenti:** [Bubbles](https://github.com/charmbracelet/bubbles) (list, table, textinput, viewport)
- **Build:** `go build` standard, nessun CGO

---

## Struttura progetto

```
lazyenv/
├── main.go                  # entrypoint, inizializza Bubble Tea
├── go.mod
├── go.sum
├── internal/
│   ├── parser/
│   │   ├── parser.go        # parsing file .env (legge, preserva commenti e ordine)
│   │   ├── writer.go        # scrittura file .env (write-back senza distruggere formato)
│   │   └── parser_test.go
│   ├── model/
│   │   ├── envfile.go       # strutture dati: EnvFile, EnvVar
│   │   └── diff.go          # engine di confronto tra due EnvFile
│   ├── tui/
│   │   ├── app.go           # model principale Bubble Tea (Init, Update, View)
│   │   ├── keys.go          # keybindings
│   │   ├── styles.go        # tema e stili Lip Gloss
│   │   ├── filelist.go      # pannello sinistro: lista file .env
│   │   ├── varlist.go       # pannello destro: lista variabili
│   │   ├── diffview.go      # vista confronto tra due file
│   │   ├── editor.go        # editing inline di valori
│   │   └── statusbar.go     # barra inferiore con comandi e messaggi
│   └── util/
│       ├── secrets.go       # detection e mascheramento valori sensibili
│       └── validator.go     # validazione valori (vuoti, placeholder, URL malformati)
└── README.md
```

---

## Strutture dati principali

```go
// EnvVar rappresenta una singola variabile
type EnvVar struct {
    Key       string
    Value     string
    Comment   string // commento inline (dopo il valore)
    LineNum   int    // posizione nel file originale
    IsSecret  bool   // rilevato come token/password/api key
    IsEmpty   bool   // valore vuoto
    IsPlaceholder bool // valore tipo "changeme", "TODO", "xxx"
}

// EnvFile rappresenta un file .env parsato
type EnvFile struct {
    Path     string
    Name     string    // nome file (es. ".env.local")
    Vars     []EnvVar
    Lines    []RawLine // tutte le righe originali (per write-back fedele)
}

// RawLine preserva il contenuto originale per riscrittura
type RawLine struct {
    Type    LineType // Variable, Comment, Empty
    Content string  // riga originale
    VarIdx  int     // indice in Vars se Type == Variable, altrimenti -1
}

// DiffEntry rappresenta una differenza tra due file
type DiffEntry struct {
    Key      string
    Status   DiffStatus // Added, Removed, Changed, Equal
    ValueA   string     // valore nel file A
    ValueB   string     // valore nel file B
}
```

---

## Funzionalita'

### MVP (fase 1)

1. **Scansione file .env**
   - All'avvio, scansiona la directory corrente (o path passato come argomento)
   - Trova tutti i file che matchano: `.env`, `.env.*`, `*.env`
   - Li mostra in un pannello navigabile

2. **Visualizzazione variabili**
   - Selezionando un file, mostra le coppie chiave=valore in una tabella
   - Colonne: chiave, valore (mascherato se segreto), flag/warning
   - Ordinamento: per posizione nel file (default) o alfabetico (toggle)

3. **Mascheramento segreti**
   - Di default, valori che matchano pattern sensibili vengono mostrati come `••••••••`
   - Pattern da rilevare: `*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, `*_PASS`, `*_API_KEY`, `*_AUTH`, `*_CREDENTIAL`
   - Toggle visibilita' con `Ctrl+S` (reveal/hide)

4. **Validazione inline**
   - Icone/colori accanto a variabili problematiche:
     - `⚠` giallo: valore vuoto
     - `⚠` arancio: valore placeholder (`changeme`, `TODO`, `xxx`, `FIXME`, `your_*_here`)
     - `⚠` rosso: duplicato (stessa chiave appare piu' volte nello stesso file)

5. **Confronto tra file (diff)**
   - Seleziona due file con `c` (compare mode)
   - Vista a due colonne con evidenziazione:
     - Verde: chiave presente solo nel file selezionato
     - Rosso: chiave mancante rispetto all'altro file
     - Giallo: stessa chiave, valore diverso
     - Grigio: uguale

6. **Editing inline**
   - `e` su una variabile: apre editor inline per modificare il valore
   - `a`: aggiunge nuova variabile (prompt chiave + valore)
   - `d`: cancella variabile (con conferma)
   - `w` o `Ctrl+S`: salva il file (write-back preservando formato)

### Fase 2 (post-MVP)

7. **Sync da template**
   - Comando `s` per sincronizzare: copia chiavi mancanti da `.env.example` a `.env`
   - Per ogni chiave mancante, chiede se inserire valore vuoto o il default dal template

8. **Genera .env.example**
   - Comando `g`: genera `.env.example` dal `.env` attuale
   - Strippa tutti i valori, lascia solo le chiavi (con commenti preservati)

9. **Ricerca globale**
   - `/` apre barra di ricerca
   - Cerca per chiave o valore in tutti i file .env trovati
   - Mostra risultati raggruppati per file

10. **Raggruppamento per prefisso**
    - Toggle con `Tab`: raggruppa variabili per prefisso comune (`DB_*`, `AWS_*`, `SMTP_*`)
    - Sezioni collassabili/espandibili

11. **Ricerca ricorsiva**
    - Flag `--recursive` o `-r`: scansiona anche sottodirectory
    - Utile per monorepo con piu' servizi

---

## Layout TUI

### Vista principale

```
┌─ Files ──────────┬─ Variables (.env) ────────────────────────────┐
│                  │                                               │
│  ● .env          │  DB_HOST          localhost                   │
│    .env.example  │  DB_PORT          5432                        │
│    .env.local    │  DB_PASSWORD      ••••••••               ⚠   │
│    .env.prod     │  DB_NAME          myapp_dev                   │
│                  │  API_KEY           ••••••••                    │
│                  │  API_URL           http://localhost:3000       │
│                  │  SMTP_HOST                                ⚠   │
│                  │  REDIS_URL         TODO                   ⚠   │
│                  │  REDIS_URL         localhost:6379         ⚠   │
│                  │                                               │
├──────────────────┴───────────────────────────────────────────────┤
│  [e]dit [a]dd [d]el [c]ompare [/]search [s]ync  ·  .env  8 vars │
└──────────────────────────────────────────────────────────────────┘
```

### Vista diff/compare

```
┌─ .env ───────────────────┬─ .env.example ────────────────────────┐
│                          │                                       │
│  DB_HOST     localhost    │  DB_HOST       localhost         =    │
│  DB_PORT     5432         │  DB_PORT       5432              =    │
│  DB_PASSWORD ••••••••     │  DB_PASSWORD   your_password     ≠    │
│  DB_NAME     myapp_dev    │  DB_NAME       myapp             ≠    │
│  API_KEY     ••••••••     │  API_KEY                         ≠    │
│  API_URL     localhost:.. │  API_URL       http://localho..  =    │
│  SMTP_HOST               │  SMTP_HOST                       =    │
│  REDIS_URL   TODO         │                                  +    │
│  REDIS_URL   localhost:.. │                                  +    │
│                          │  LOG_LEVEL     debug              -    │
│                          │  NODE_ENV      development        -    │
│                          │                                       │
├──────────────────────────┴───────────────────────────────────────┤
│  = equal  ≠ different  + only left  - only right  ·  6 diffs     │
└──────────────────────────────────────────────────────────────────┘
```

---

## Keybindings

| Tasto        | Azione                                    |
|-------------|-------------------------------------------|
| `↑/↓` `j/k` | Naviga nella lista                       |
| `←/→` `h/l` | Switch tra pannello file e variabili     |
| `Enter`      | Seleziona file / espandi dettaglio       |
| `e`          | Edit valore della variabile selezionata  |
| `a`          | Aggiungi nuova variabile                 |
| `d`          | Cancella variabile (con conferma)        |
| `c`          | Entra in compare mode (seleziona 2 file) |
| `Escape`     | Esci da compare/edit/search mode         |
| `/`          | Apri barra di ricerca                    |
| `s`          | Sync: importa chiavi mancanti da template|
| `g`          | Genera .env.example                      |
| `Ctrl+S`     | Toggle visibilita' segreti               |
| `Tab`        | Toggle raggruppamento per prefisso       |
| `o`          | Toggle ordinamento (posizione/alfabetico)|
| `w`          | Salva modifiche                          |
| `q`          | Esci                                     |
| `?`          | Mostra help keybindings                  |

---

## Parser .env

Il parser deve gestire correttamente:

```bash
# Commento a riga intera
DB_HOST=localhost              # commento inline

# Valori con/senza virgolette
SIMPLE=value
QUOTED="value with spaces"
SINGLE_QUOTED='value'

# Valori multilinea (tra virgolette doppie)
MULTI="line1
line2
line3"

# Valori vuoti
EMPTY=
ALSO_EMPTY=""

# Export prefix (ignorare "export")
export NODE_ENV=production

# Righe vuote preservate per write-back
```

Regole:
- Preservare ordine righe, commenti e righe vuote (per write-back fedele)
- Strippare `export ` prefix se presente
- Gestire quoting: singolo, doppio, nessuno
- Gestire escape: `\"`, `\\`, `\n` dentro virgolette doppie
- Ignorare righe che non matchano `KEY=VALUE`
- Rilevare duplicati (stessa chiave piu' volte)

---

## Detection segreti

Logica per determinare se una variabile contiene un segreto:

```go
// Match per suffisso chiave (case-insensitive)
var secretSuffixes = []string{
    "_KEY", "_SECRET", "_TOKEN", "_PASSWORD", "_PASS",
    "_API_KEY", "_AUTH", "_CREDENTIAL", "_PRIVATE",
}

// Match per prefisso chiave
var secretPrefixes = []string{
    "SECRET_", "TOKEN_", "AUTH_", "PRIVATE_",
}

// Match esatto
var secretExact = []string{
    "PASSWORD", "SECRET", "TOKEN", "API_KEY",
    "ACCESS_KEY", "PRIVATE_KEY",
}

// Match per pattern valore (sembra un token/hash)
// - Lunghezza >= 20 e contiene mix alfanumerico
// - Inizia con prefissi noti: "sk-", "pk-", "ghp_", "gho_", "Bearer "
```

---

## CLI

```
Usage: lazyenv [path] [flags]

Arguments:
  path    Directory da scansionare (default: directory corrente)

Flags:
  -r, --recursive    Scansiona anche sottodirectory
  -a, --show-all     Mostra anche segreti in chiaro all'avvio
  -h, --help         Mostra help
  -v, --version      Mostra versione
```

---

## Note di implementazione

- Usare `cobra` o parsing flag manuale (lo scope e' minimo, cobra potrebbe essere overkill)
- Il parser deve essere un package separato e ben testato: e' il cuore dell'app
- Il write-back deve produrre output identico all'input se non ci sono modifiche (round-trip fedelta')
- I test del parser devono coprire: commenti, quoting, escape, multiline, duplicati, export prefix, righe vuote
- Lo stato della TUI dovrebbe essere un singolo struct che implementa `tea.Model`
- Separare la logica di rendering (View) dalla logica di business (Update)
- Lip Gloss adaptive colors per supportare sia temi chiari che scuri del terminale

---

## Milestone suggerite

1. **M1 — Parser e strutture dati**
   - Parser .env completo con test
   - Writer con round-trip fedelta'
   - Strutture dati EnvFile, EnvVar

2. **M2 — TUI base**
   - Layout a due pannelli (file list + variable list)
   - Navigazione con tastiera
   - Mascheramento segreti
   - Validazione inline (vuoti, placeholder, duplicati)

3. **M3 — Diff e confronto**
   - Compare mode tra due file
   - Vista diff a due colonne
   - Evidenziazione differenze

4. **M4 — Editing**
   - Edit valore inline
   - Aggiungi/cancella variabile
   - Salvataggio con write-back fedele

5. **M5 — Feature avanzate**
   - Sync da template
   - Genera .env.example
   - Ricerca globale
   - Raggruppamento per prefisso
