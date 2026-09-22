package main

// winres schreibt die Windows-Ressourcen einer Starter-EXE – Icon, Manifest
// und Versionsangaben – als .syso-Datei, die go build von selbst einbindet.
// Das Icon kommt aus einer .ico-Datei oder aus einer anderen .exe.

import (
	"bytes"
	"cmp"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtVersion   = 16
	rtManifest  = 24
	langENUS    = 0x0409
)

var le = binary.LittleEndian

// resInfo beschreibt, was in die Ressourcen einer Starter-EXE kommt.
type resInfo struct {
	images      []iconImage
	version     [4]uint16
	description string // FileDescription – so heißt die EXE in Taskleiste und Task-Manager
	filename    string // OriginalFilename
}

// writeSyso schreibt Icon, Versionsangaben und Manifest nach out.
func writeSyso(info resInfo, out string) error {
	var res []resource
	group := make([]byte, 6, 6+14*len(info.images)) // GRPICONDIR
	le.PutUint16(group[2:], 1)
	le.PutUint16(group[4:], uint16(len(info.images)))
	for i, img := range info.images {
		id := uint16(i + 1)
		res = append(res, resource{rtIcon, id, img.data})
		entry := make([]byte, 14) // GRPICONDIRENTRY
		copy(entry, img.meta[:])
		le.PutUint32(entry[8:], uint32(len(img.data)))
		le.PutUint16(entry[12:], id)
		group = append(group, entry...)
	}
	res = append(res,
		resource{rtGroupIcon, 1, group},
		resource{rtVersion, 1, versionInfo(info)},
		resource{rtManifest, 1, []byte(manifest(info.version))},
	)
	return os.WriteFile(out, coff(res), 0o644)
}

func parseVersion(s string) ([4]uint16, error) {
	var v [4]uint16
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) > 4 {
		return v, fmt.Errorf("Version %q: erwartet z. B. 1.0.0", s)
	}
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return v, fmt.Errorf("Version %q: erwartet z. B. 1.0.0", s)
		}
		v[i] = uint16(n)
	}
	return v, nil
}

// iconImage ist ein Bild des Icons: die ersten 12 Bytes seines
// Verzeichniseintrags (Breite, Höhe, Farben, Ebenen, Bits, Größe) und die Daten.
type iconImage struct {
	meta [12]byte
	data []byte
}

func loadIcon(path string) ([]iconImage, error) {
	if strings.EqualFold(filepath.Ext(path), ".exe") {
		return iconFromEXE(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return iconFromICO(data)
}

func iconFromICO(b []byte) ([]iconImage, error) {
	if len(b) < 6 || le.Uint16(b[0:]) != 0 || le.Uint16(b[2:]) != 1 {
		return nil, errors.New("keine .ico-Datei")
	}
	n := int(le.Uint16(b[4:]))
	if n == 0 || len(b) < 6+16*n {
		return nil, errors.New("beschädigte .ico-Datei")
	}
	images := make([]iconImage, n)
	for i := range images {
		e := b[6+16*i:]
		size, off := uint64(le.Uint32(e[8:])), uint64(le.Uint32(e[12:]))
		if off+size > uint64(len(b)) {
			return nil, errors.New("beschädigte .ico-Datei")
		}
		copy(images[i].meta[:], e[:12])
		images[i].data = b[off : off+size]
	}
	return images, nil
}

// iconFromEXE holt das Icon, das der Explorer für die .exe zeigt: die erste
// Icon-Gruppe in deren Ressourcen.
func iconFromEXE(path string) ([]iconImage, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := openResources(f)
	if err != nil {
		return nil, err
	}
	group, err := r.find(rtGroupIcon, -1)
	if err != nil {
		return nil, err
	}
	if len(group) < 6 {
		return nil, errors.New("beschädigte Icon-Gruppe")
	}
	n := int(le.Uint16(group[4:]))
	if n == 0 || len(group) < 6+14*n {
		return nil, errors.New("beschädigte Icon-Gruppe")
	}
	images := make([]iconImage, n)
	for i := range images {
		e := group[6+14*i:]
		data, err := r.find(rtIcon, int(le.Uint16(e[12:])))
		if err != nil {
			return nil, err
		}
		copy(images[i].meta[:], e[:12])
		images[i].data = data
	}
	return images, nil
}

// resources liest das Ressourcenverzeichnis einer .exe.
type resources struct {
	data []byte // Inhalt des Abschnitts mit den Ressourcen
	va   uint32 // dessen virtuelle Adresse
	root uint32 // virtuelle Adresse des Verzeichnisses
}

func openResources(f *pe.File) (*resources, error) {
	var dirs []pe.DataDirectory
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		dirs = oh.DataDirectory[:min(oh.NumberOfRvaAndSizes, uint32(len(oh.DataDirectory)))]
	case *pe.OptionalHeader64:
		dirs = oh.DataDirectory[:min(oh.NumberOfRvaAndSizes, uint32(len(oh.DataDirectory)))]
	}
	if len(dirs) <= pe.IMAGE_DIRECTORY_ENTRY_RESOURCE || dirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress == 0 {
		return nil, errors.New("keine Ressourcen")
	}
	root := dirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	for _, s := range f.Sections {
		if root >= s.VirtualAddress && root < s.VirtualAddress+max(s.VirtualSize, s.Size) {
			data, err := s.Data()
			if err != nil {
				return nil, err
			}
			return &resources{data: data, va: s.VirtualAddress, root: root}, nil
		}
	}
	return nil, errors.New("Ressourcen nicht gefunden")
}

// at liefert size Bytes ab der virtuellen Adresse addr.
func (r *resources) at(addr, size uint32) ([]byte, error) {
	start := int64(addr) - int64(r.va)
	if start < 0 || start+int64(size) > int64(len(r.data)) {
		return nil, errors.New("Ressource außerhalb ihres Abschnitts")
	}
	return r.data[start : start+int64(size)], nil
}

type dirEntry struct {
	id     uint32
	named  bool
	subdir bool
	offset uint32 // ab Verzeichnisanfang
}

func (r *resources) dir(offset uint32) ([]dirEntry, error) {
	hdr, err := r.at(r.root+offset, 16)
	if err != nil {
		return nil, err
	}
	n := uint32(le.Uint16(hdr[12:])) + uint32(le.Uint16(hdr[14:]))
	raw, err := r.at(r.root+offset+16, 8*n)
	if err != nil {
		return nil, err
	}
	entries := make([]dirEntry, n)
	for i := range entries {
		name, target := le.Uint32(raw[8*i:]), le.Uint32(raw[8*i+4:])
		entries[i] = dirEntry{id: name & 0xffff, named: name>>31 == 1, subdir: target>>31 == 1, offset: target & 0x7fffffff}
	}
	return entries, nil
}

// find liefert die Daten der Ressource typ/id in der ersten Sprache; id < 0
// nimmt die erste Ressource des Typs.
func (r *resources) find(typ uint16, id int) ([]byte, error) {
	pick := func(entries []dirEntry, id int) (dirEntry, bool) {
		for _, e := range entries {
			if e.subdir && (id < 0 || (!e.named && int(e.id) == id)) {
				return e, true
			}
		}
		return dirEntry{}, false
	}
	types, err := r.dir(0)
	if err != nil {
		return nil, err
	}
	t, ok := pick(types, int(typ))
	if !ok {
		return nil, fmt.Errorf("kein Ressourcentyp %d", typ)
	}
	names, err := r.dir(t.offset)
	if err != nil {
		return nil, err
	}
	n, ok := pick(names, id)
	if !ok {
		return nil, fmt.Errorf("Ressource %d/%d fehlt", typ, id)
	}
	langs, err := r.dir(n.offset)
	if err != nil {
		return nil, err
	}
	if len(langs) == 0 || langs[0].subdir {
		return nil, fmt.Errorf("Ressource %d/%d beschädigt", typ, id)
	}
	entry, err := r.at(r.root+langs[0].offset, 16) // IMAGE_RESOURCE_DATA_ENTRY
	if err != nil {
		return nil, err
	}
	return r.at(le.Uint32(entry[0:]), le.Uint32(entry[4:]))
}

// versionInfo baut VS_VERSIONINFO – der Explorer zeigt es unter Eigenschaften › Details.
func versionInfo(info resInfo) []byte {
	v := info.version
	ms, ls := uint32(v[0])<<16|uint32(v[1]), uint32(v[2])<<16|uint32(v[3])
	fixed := make([]byte, 52) // VS_FIXEDFILEINFO
	for i, x := range []uint32{0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x00040004, 1, 0, 0, 0} {
		le.PutUint32(fixed[4*i:], x)
	}
	version := fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
	var strs [][]byte
	for _, kv := range [][2]string{
		{"CompanyName", "sorglos-it"},
		{"FileDescription", info.description},
		{"FileVersion", version},
		{"InternalName", strings.TrimSuffix(info.filename, ".exe")},
		{"LegalCopyright", "© Sorglos Thomas Weirich, MIT-Lizenz"},
		{"OriginalFilename", info.filename},
		{"ProductName", "Portable Starter"},
		{"ProductVersion", version},
	} {
		value := utf16z(kv[1])
		strs = append(strs, node(kv[0], 1, value, uint16(len(value)/2)))
	}
	stringInfo := node("StringFileInfo", 1, nil, 0, node("040904B0", 1, nil, 0, strs...))
	varInfo := node("VarFileInfo", 1, nil, 0, node("Translation", 0, []byte{0x09, 0x04, 0xB0, 0x04}, 4))
	return node("VS_VERSION_INFO", 0, fixed, 52, stringInfo, varInfo)
}

// node baut einen Knoten der Versionsangaben: Länge, Wertlänge, Typ,
// Schlüssel, Wert und Kinder, jeweils auf 4 Bytes ausgerichtet.
func node(key string, typ uint16, value []byte, valueLen uint16, children ...[]byte) []byte {
	b := make([]byte, 6)
	le.PutUint16(b[2:], valueLen)
	le.PutUint16(b[4:], typ)
	b = pad4(append(b, utf16z(key)...))
	b = append(b, value...)
	for _, c := range children {
		b = append(pad4(b), c...)
	}
	le.PutUint16(b[0:], uint16(len(b)))
	return b
}

func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

func utf16z(s string) []byte {
	u := append(utf16.Encode([]rune(s)), 0)
	b := make([]byte, 2*len(u))
	for i, c := range u {
		le.PutUint16(b[2*i:], c)
	}
	return b
}

// manifest: normale Rechte, Windows 10/11, gestochen scharf bei hoher
// Skalierung, Dialoge im aktuellen Windows-Aussehen.
func manifest(v [4]uint16) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="sorglos-it.portable-starter" version="%d.%d.%d.%d" processorArchitecture="*"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
    </windowsSettings>
  </application>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
</assembly>
`, v[0], v[1], v[2], v[3])
}

type resource struct {
	typ, id uint16
	data    []byte
}

// coff verpackt die Ressourcen als COFF-Objekt für amd64 mit einem Abschnitt
// .rsrc; der Linker setzt die Adressen über die Relocations ein.
func coff(res []resource) []byte {
	sec, relocs := rsrcSection(res)
	const headers = 20 + 40 // IMAGE_FILE_HEADER + IMAGE_SECTION_HEADER
	var name [8]uint8
	copy(name[:], ".rsrc")
	var b bytes.Buffer
	binary.Write(&b, le, pe.FileHeader{
		Machine:              pe.IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     1,
		PointerToSymbolTable: uint32(headers + len(sec) + 10*len(relocs)),
		NumberOfSymbols:      1,
	})
	binary.Write(&b, le, pe.SectionHeader32{
		Name:                 name,
		SizeOfRawData:        uint32(len(sec)),
		PointerToRawData:     headers,
		PointerToRelocations: uint32(headers + len(sec)),
		NumberOfRelocations:  uint16(len(relocs)),
		Characteristics:      0x40000040, // IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
	})
	b.Write(sec)
	for _, off := range relocs {
		binary.Write(&b, le, pe.Reloc{VirtualAddress: off, Type: 3}) // IMAGE_REL_AMD64_ADDR32NB
	}
	binary.Write(&b, le, pe.COFFSymbol{Name: name, SectionNumber: 1, StorageClass: 3}) // IMAGE_SYM_CLASS_STATIC
	binary.Write(&b, le, uint32(4))                                                    // leere Stringtabelle
	return b.Bytes()
}

// rsrcSection baut den Abschnitt .rsrc: Verzeichnisse Typ → ID → Sprache,
// dann die Dateneinträge, dann die Daten. relocs sind die Stellen, an denen
// Adressen relativ zum Abschnitt stehen.
func rsrcSection(res []resource) (sec []byte, relocs []uint32) {
	res = slices.Clone(res)
	slices.SortFunc(res, func(a, b resource) int { return cmp.Or(cmp.Compare(a.typ, b.typ), cmp.Compare(a.id, b.id)) })
	var types []uint16
	count := map[uint16]int{}
	for _, r := range res {
		if count[r.typ] == 0 {
			types = append(types, r.typ)
		}
		count[r.typ]++
	}

	off := 16 + 8*len(types)
	typeDir := map[uint16]int{}
	for _, t := range types {
		typeDir[t] = off
		off += 16 + 8*count[t]
	}
	langDir, dataEntry, dataAt := make([]int, len(res)), make([]int, len(res)), make([]int, len(res))
	for i := range res {
		langDir[i] = off
		off += 16 + 8
	}
	for i := range res {
		dataEntry[i] = off
		off += 16
	}
	for i, r := range res {
		off = (off + 7) &^ 7
		dataAt[i] = off
		off += len(r.data)
	}
	sec = make([]byte, (off+7)&^7)

	dirHeader := func(at, entries int) { le.PutUint16(sec[at+14:], uint16(entries)) }
	dirEntry := func(at int, id uint32, target uint32) {
		le.PutUint32(sec[at:], id)
		le.PutUint32(sec[at+4:], target)
	}
	dirHeader(0, len(types))
	for i, t := range types {
		dirEntry(16+8*i, uint32(t), 1<<31|uint32(typeDir[t]))
	}
	next := map[uint16]int{}
	for i, r := range res {
		dirHeader(typeDir[r.typ], count[r.typ])
		dirEntry(typeDir[r.typ]+16+8*next[r.typ], uint32(r.id), 1<<31|uint32(langDir[i]))
		next[r.typ]++

		dirHeader(langDir[i], 1)
		dirEntry(langDir[i]+16, langENUS, uint32(dataEntry[i]))

		le.PutUint32(sec[dataEntry[i]:], uint32(dataAt[i])) // wird zur RVA
		le.PutUint32(sec[dataEntry[i]+4:], uint32(len(r.data)))
		relocs = append(relocs, uint32(dataEntry[i]))
		copy(sec[dataAt[i]:], r.data)
	}
	return sec, relocs
}
