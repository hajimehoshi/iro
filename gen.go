// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

//go:build ignore

// This program generates munselltable.go from the copies of the Munsell renotation
// data in munselldata. See the Notice section of README.md for their sources.
//
// Usage:
//
//	go generate ./...        regenerate from the copies in munselldata
//	go run gen.go -download  refresh those copies from their sources first
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const dataDir = "munselldata"

// The renotation data is published as two files, which are read in this order.
// all.dat covers the whole grid, including points extrapolated beyond real surface
// colours, but rounds some entries to as few as two decimals. real.dat covers only
// the real colours and is uniformly given to four decimals, so it takes precedence
// where it applies.
var sources = []struct {
	name string
	url  string
}{
	{"all.dat", "https://www.rit-mcsl.org/MunsellRenotation/all.dat"},
	{"real.dat", "https://www.rit-mcsl.org/MunsellRenotation/real.dat"},
}

var download = flag.Bool("download", false, "refresh the copies in "+dataDir+" from their sources")

// hueBases maps a Munsell hue family to its base position on the ASTM hue
// scale, on which the 40 renotation hues are 2.5, 5, ..., 100.
var hueBases = map[string]float64{
	"R":  0,
	"YR": 10,
	"Y":  20,
	"GY": 30,
	"G":  40,
	"BG": 50,
	"B":  60,
	"PB": 70,
	"P":  80,
	"RP": 90,
}

// values lists the Munsell values covered by the renotation data, in order.
var values = []float64{0.2, 0.4, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

const hueCount = 40

type entry struct {
	x, y float64
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// groups[hueIndex*len(values)+valueIndex][chromaIndex] holds the chromaticity
	// for one (hue, value) pair, indexed by chroma/2-1.
	groups := make([]map[int]entry, hueCount*len(values))
	for i := range groups {
		groups[i] = map[int]entry{}
	}

	valueIndices := map[float64]int{}
	for i, v := range values {
		valueIndices[v] = i
	}

	if *download {
		for _, src := range sources {
			if err := refresh(src.name, src.url); err != nil {
				return err
			}
		}
	}

	var rows int
	for _, src := range sources {
		n, err := collect(src.name, groups, valueIndices)
		if err != nil {
			return err
		}
		// Only the first pass adds entries; the second refines those it covers.
		if rows == 0 {
			rows = n
		}
	}

	return emit(groups, rows)
}

// refresh replaces the copy of name in dataDir with the contents of url.
func refresh(name, url string) error {
	data, err := fetch(url)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dataDir, name), data, 0o644); err != nil {
		return err
	}

	fmt.Printf("%s: %d bytes from %s\n", name, len(data), url)
	return nil
}

// collect parses the copy of the renotation data in name into groups, overwriting
// any entry it covers, and returns the number of rows read.
func collect(name string, groups []map[int]entry, valueIndices map[float64]int) (int, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, name))
	if err != nil {
		return 0, err
	}

	var rows int
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 6 {
			continue
		}
		// Skip the header line.
		if fields[0] == "H" || fields[0] == "h" {
			continue
		}

		hue, err := parseHue(fields[0])
		if err != nil {
			return 0, err
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, err
		}
		chroma, err := strconv.Atoi(fields[2])
		if err != nil {
			return 0, err
		}
		x, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			return 0, err
		}
		y, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			return 0, err
		}

		vi, ok := valueIndices[value]
		if !ok {
			return 0, fmt.Errorf("gen: unexpected value %v", value)
		}
		if chroma < 2 || chroma%2 != 0 {
			return 0, fmt.Errorf("gen: unexpected chroma %d", chroma)
		}

		hi := int(hue/2.5) - 1
		groups[hi*len(values)+vi][chroma/2-1] = entry{x: x, y: y}
		rows++
	}
	if err := s.Err(); err != nil {
		return 0, err
	}
	return rows, nil
}

// emit flattens the groups and writes munselltable.go.
func emit(groups []map[int]entry, rows int) error {
	// The chroma runs are contiguous from 2 upwards, so each group is stored as a
	// flat run and located by a prefix-sum offset table.
	var xy []entry
	offsets := make([]int, len(groups)+1)
	for g, m := range groups {
		offsets[g] = len(xy)
		for c := 0; c < len(m); c++ {
			e, ok := m[c]
			if !ok {
				return fmt.Errorf("gen: group %d has a gap at chroma %d", g, (c+1)*2)
			}
			xy = append(xy, e)
		}
	}
	offsets[len(groups)] = len(xy)

	if len(xy) != rows {
		return fmt.Errorf("gen: got %d entries, want %d", len(xy), rows)
	}

	src, err := generate(xy, offsets, rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile("munselltable.go", src, 0o644); err != nil {
		return err
	}

	fmt.Printf("munselltable.go: %d entries, %d bytes\n", rows, len(src))
	return nil
}

func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gen: fetching %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// parseHue converts a renotation hue such as "7.5PB" to its ASTM hue number.
func parseHue(s string) (float64, error) {
	i := strings.IndexFunc(s, func(r rune) bool {
		return r != '.' && (r < '0' || r > '9')
	})
	if i <= 0 {
		return 0, fmt.Errorf("gen: invalid hue %q", s)
	}

	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, fmt.Errorf("gen: invalid hue %q: %w", s, err)
	}
	base, ok := hueBases[s[i:]]
	if !ok {
		return 0, fmt.Errorf("gen: unknown hue family in %q", s)
	}
	return base + n, nil
}

func generate(xy []entry, offsets []int, rows int) ([]byte, error) {
	var b bytes.Buffer

	fmt.Fprint(&b, `// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

// Code generated by gen.go. DO NOT EDIT.

package iro

`)
	fmt.Fprintf(&b, "// munsellXY holds the %d chromaticity pairs of the Munsell renotation data,\n", rows)
	fmt.Fprint(&b, "// grouped by (hue, value) and ordered by chroma within each group.\n")
	fmt.Fprint(&b, "var munsellXY = [...]float64{\n")
	for i, e := range xy {
		if i%4 == 0 {
			b.WriteString("\t")
		}
		fmt.Fprintf(&b, "%s, %s,", formatFloat(e.x), formatFloat(e.y))
		if i%4 == 3 || i == len(xy)-1 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	fmt.Fprint(&b, "}\n\n")

	fmt.Fprint(&b, "// munsellOffsets locates each (hue, value) group in munsellXY as the half-open\n")
	fmt.Fprint(&b, "// range [munsellOffsets[g], munsellOffsets[g+1]) of chromaticity pairs.\n")
	fmt.Fprint(&b, "var munsellOffsets = [...]uint16{\n")
	for i, o := range offsets {
		if i%12 == 0 {
			b.WriteString("\t")
		}
		fmt.Fprintf(&b, "%d,", o)
		if i%12 == 11 || i == len(offsets)-1 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	fmt.Fprint(&b, "}\n")

	return format.Source(b.Bytes())
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
