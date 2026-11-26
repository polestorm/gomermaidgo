# Gallery

Every picture here is the SVG this library writes for the source next to it, with only a white background
added (`WithBackground("white")`) so it reads well in a dark theme too. The sources are in [`src/`](src/); the
pictures are made again with `go run ./tools/gallery`.

- [Page fitting](page-fitting.md): the same diagram with and without fitting to A4, folding into rows,
  folding of long chains and staggering of long ranks.
- [Diagram types](types.md): one diagram of each of the 28 types.
- [Scripts and symbols](scripts.md): Polish, German, French, Greek, Cyrillic, emoji and symbols with the bundled
  fonts, and Japanese with a font given by the caller.

The examples come from quality management, the field this library was written for, but nothing in the
library is specific to it.
