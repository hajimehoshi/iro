# Iro (色)

Package iro provides color conversion utilities between various color spaces.

Unlike the standard library’s `color.Color`, it keeps values in the XYZ D65 space to minimize loss when moving between spaces (within floating-point limits).

## Usage

```go
package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/iro"
)

func main() {
	// Convert nonlinear sRGB to OKLch.
	c := iro.ColorFromSRGB(0.2, 0.4, 0.6, 1)
	l, ch, h, alpha := c.OKLch()
	hDeg := h * 180 / math.Pi

	fmt.Printf("L=%.6f C=%.6f h=%.2f° alpha=%.2f\n", l, ch, hDeg, alpha)
}
```
