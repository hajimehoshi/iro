// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package iro

// MunsellChromaticity exposes munsellChromaticity for tests, which compare against
// the renotation data in its own illuminant C reference.
var MunsellChromaticity = munsellChromaticity

// MunsellLuminance exposes munsellLuminance for tests.
var MunsellLuminance = munsellLuminance

// IlluminantC is the chromaticity the renotation data is referred to.
const (
	IlluminantCx = illuminantCx
	IlluminantCy = illuminantCy
)
