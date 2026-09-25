# Portable Starter

[![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Donate with PayPal](https://img.shields.io/badge/Donate-PayPal-00457C?logo=paypal&logoColor=white)](https://www.paypal.com/donate/?hosted_button_id=6CDEVZGJWTNQQ)

One starter for all portable Windows programs. It starts the program so that
its settings stay in the program folder instead of the user profile.
Which program runs with which parameters is set in `config_starter.json`.

| Folder | Purpose | Language | Start | Build |
| --- | --- | --- | --- | --- |
| `apps/desktop` | the starter, `starter.exe` | Go | `dist\starter.exe` | `tools\build.bat` |

## Start in 3 steps

1. Copy the matching starter from `dist\` into the program's folder, e.g.
   `portable_drawio.exe` next to `draw.io.exe`. For other programs take `starter.exe` –
   any name will do.
2. Double-click it. On the first start the starter creates `config_starter.json`.
3. If the program is not listed there, enter its EXE and parameters – the starter offers to open the file.

Already listed, and ready in `dist\` with their own icon: draw.io, OrcaSlicer,
Creality Print, PrusaSlicer, QElectroTech, Mullvad Browser and ecoDMS. Coming from
`OrcaSlicerPortableStarter`, `CrealityPrintPortableStarter` or `PrusaSlicerPortable`?
Your `profile` folder stays in use.

**Update:** copy the new EXE over the old one. `config_starter.json` and the data stay.

## Two layouts

```text
Everything in one folder       Program one level down
drawio\                        drawio\
├── draw.io.exe                ├── app\                ← program, replaced on update
├── portable_drawio.exe        │   └── draw.io.exe
├── config_starter.json        ├── daten\              ← settings
└── daten\                     ├── lib\                ← if needed
                               ├── portable_drawio.exe
                               └── config_starter.json
```

The starter looks for the EXE next to itself, otherwise in the subfolder `app`. The program
runs in its own folder, like on a double-click; the data stay next to the starter –
an update only replaces `app\`.

**Moving to the second layout:** move the program files into `app\`; starter, JSON and
data folder stay on top. Then double-click `tools\verknuepfungen.bat` and pick the program
folder: Start menu entries, file types and uninstall entries that now point nowhere are
redirected to the starter or to the new place in `app\`. The tool shows everything first,
backs it up in your Documents folder and asks before it changes anything.

## config_starter.json

The key names are German: `programme` (programs), `vorher` (before), `warten` (wait);
the placeholders `{ordner}` (folder) and `{daten}` (data).

```json
{
  "programme": [
    {
      "exe": "draw.io.exe",
      "parameter": ["--user-data-dir={daten}", "--disable-update"]
    },
    {
      "exe": "orca-slicer.exe",
      "parameter": ["--datadir", "{ordner}/profile"]
    }
  ]
}
```

- The starter takes the **first** program in the list that is in its folder.
  So one file fits many programs.
- `exe` – file name, also with a subfolder: `bin/program.exe`.
- `parameter` – one entry per parameter. Spaces need no quotation marks.
- For data next to the starter use `{ordner}` or `{daten}`: `"--datadir", "{ordner}/profile"`.
- `{ordner}` – the starter's folder. `{daten}` – its subfolder `daten`, created if missing.
- `{app}` – the folder with the program files: `app\`, otherwise the starter's folder.
- Files dropped onto the starter are passed on to the program.

If a program needs another one running first, `vorher` starts that one first;
`warten` is the pause afterwards in seconds (0 to 60):

```json
{
  "exe": "ecodmsclient.exe",
  "vorher": [
    { "exe": "ecodmssinglesignon.exe", "warten": 3 }
  ]
}
```

Mistakes in the file are reported with line and character.

## Starter with the program's icon

Drag program folders onto `tools\build.bat`, several at once if you like:

```bat
tools\build.bat "D:\Apps\drawio" "D:\Apps\OrcaSlicer"
```

Each folder gets `portable_<program>.exe` with the icon and name of the
program the starter finds there – if `config_starter.json` is missing, it is
added. If the EXE has no icon, the tool takes an `.ico` of the same name from the
program folder, otherwise the default icon. A copy goes to `dist\`.

## Build

The only requirement is [Go](https://go.dev/dl/) 1.24 or later.
`tools\build.bat` without arguments checks, tests and puts `starter.exe` and
`config_starter.json` into `dist\`. `dist\` is part of the repo – to just
start programs, you don't need Go.

## Security

- Starts only `.exe` files and never itself.
- Runs with normal rights. If a program needs admin rights, Windows asks – as with a double-click.
- Loads system libraries only from `System32`, never from the program folder.
- No network, no registry. The starter exits as soon as the program is running.

## Donate via PayPal

If the starter saves you work, a donation is very welcome. Thank you!

[![Donate with PayPal](https://img.shields.io/badge/Donate-PayPal-00457C?logo=paypal&logoColor=white&style=for-the-badge)](https://www.paypal.com/donate/?hosted_button_id=6CDEVZGJWTNQQ)

## License

[MIT](LICENSE). Not affiliated with the makers of the programs it starts –
their names and icons in `dist\portable_*.exe` belong to them.
