# Syntax profile

The supported range of Mermaid syntax. Anything not listed in a "Supported" table is an error with a line number.

## Common to all types

- A YAML front matter block between `---` lines is accepted; `title` gives a title above the drawing (COM-001).
- The `%%{init: … }%%` directive, including a multi-line one, is accepted (COM-002).
- From the front matter and the directive, the theme (`theme`), four theme variables (`primaryColor`,
  `lineColor`, `primaryTextColor`, `background`) and the text wrapping settings (COM-008) are read; the rest of the
  configuration is ignored, see the "Mermaid configuration" table below. The themes are this library's own, under
  Mermaid's names: `neutral` (the default), `default` (colours), `forest`, `dark`, `base`. The caller can impose a theme
  (`WithTheme`, the `-t` flag), which disables the choice made in the source (THM-001…004, ADR 0005).
- Line numbers in error messages count from the start of the source, front matter included (COM-003).

### Mermaid configuration

What happens to configuration keys written in the source (a `---` block with `config:`, or `%%{init}%%`). Keys not in
the table are ignored without an error. Library call options (`WithTheme`, `WithFont`, `WithFit` and so on) take
precedence over the source where both set the same thing.

| Mermaid key | Here | Notes |
|---|---|---|
| `theme`, `themeVariables` (`primaryColor`, `lineColor`, `primaryTextColor`, `background`) | supported | own themes under Mermaid's names (THM-001…004) |
| `markdownAutoWrap`, `wrap` | supported | `false` turns off automatic text wrapping in graph diagrams; `wrap: true` wraps texts in a sequence diagram (COM-008) |
| `flowchart.wrappingWidth` | supported | the wrapping limit for text in fields and on edges (COM-008) |
| `darkMode` | through the theme | equivalent: `theme: dark` |
| `fontFamily`, `fontSize` | only from the call | `WithFont`, `WithFontSize`; the source does not change the document's typeface |
| `layout` (`dagre`, `elk`), `elk.*` | ignored | own layered layout; own directives `%% layout: swimlanes`, `fit`/`nofit`, `fold`/`nofold`, `fold-chains`, `fold-ranks` |
| `look` (`handDrawn`), `handDrawnSeed` | ignored | a single drawing style |
| `flowchart.nodeSpacing`, `rankSpacing`, `padding`, `diagramPadding`, `subGraphTitleMargin`, `curve` | ignored | fixed spacing, relative to the font; curves: `WithCurvedEdges` |
| `maxTextSize`, `maxEdges` | ignored | own fixed limits; the source is external data and cannot raise them |
| `htmlLabels`, `securityLevel`, `startOnLoad`, `logLevel`, `deterministicIds`, `deterministicIDSeed`, `themeCSS`, `suppressErrorRendering`, `arrowMarkerAbsolute`, `secure`, `legacyMathML`, `forceLegacyMathML` | not applicable | they concern the browser; the output has no HTML, CSS, scripts or random identifiers |
| diagram type keys (`sequence.*`, `gantt.*`, `pie.*` etc.) | ignored | except those listed (`sequence.wrap`); ignored keys are listed under the types below |

## Sequence diagram (`sequenceDiagram`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, comments | `sequenceDiagram`, `%% komentarz` | SEQ-001 |
| participant, actor, alias | `participant A`, `actor A`, `participant A as Etykieta` | SEQ-002 |
| implicit participant | first use in a message or a note | SEQ-002 |
| message | `A->>B: tekst`; arrows `->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`, `<<->>`, `<<-->>` | SEQ-003 |
| message to self | `A->>A: tekst` | SEQ-003 |
| note | `Note right of A: t`, `Note left of A: t`, `Note over A: t`, `Note over A,B: t` | SEQ-004 |
| activation | `activate A`, `deactivate A`, `A->>+B: t`, `B-->>-A: t` | SEQ-005 |
| blocks | `loop`, `alt`/`else`, `opt`, `par`/`and`, `critical`/`option`, `break`, closed with `end`; nested freely | SEQ-006, SEQ-007 |
| background highlight | `rect rgb(r, g, b)`, `rect rgba(r, g, b, a)`, `rect #rrggbb` … `end` | SEQ-007 |
| numbering | `autonumber`, `autonumber 10`, `autonumber 10 5` (start, step) | SEQ-008 |
| participant groups | `box [kolor] [tytuł]` … `end`; colour: `rgb()`, `rgba()`, `#hex`, a basic colour name, `transparent` | SEQ-014 |
| creating and destroying a participant | `create participant A`, `create actor A`, `destroy A` (each before the message it applies to) | SEQ-015 |
| accessibility | `accTitle: tekst`, `accDescr: tekst`, `accDescr { … }` → `title` and `desc` in the SVG | SEQ-016 |
| title | `title Tekst` | SEQ-010 |
| line break in text | `<br/>`, `<br>` | SEQ-010 |

### Outside the profile (error `sequence-unsupported`)

`link`, `links`, `properties`, `details`.

### Deliberate differences from Mermaid

- a semicolon does not separate statements and entity codes (`#35;`) are not expanded: the text goes into the drawing
  verbatim;
- text wrapping only for the whole diagram (`wrap: true` in the configuration, COM-008), without the `wrap:` prefix in
  the message text; `%%{init}%%` directives and the `---` block work as for all types (see above);
- limits: 50 participants, 2000 statements, 20 levels of block nesting;
- deactivating an inactive participant is an error, in the shorthand form too;
- a participant ended by `destroy` ends with a cross and has no footer; Mermaid draws its box a second time at that
  point. When the ending message has a cross arrowhead (`-x`), the arrowhead and the end mark are a single cross.
  Using such a participant after it has ended is an error;
- the label of a message to self stands above the loop; for the last participant it ends above the loop and grows to
  the left, so as not to widen the drawing;
- the look is this library's own; themes and theme variables are not supported. The criterion for the look is
  legibility, not conformance to UML notation: block frames (`alt`, `loop`, `opt` and the others) are dotted so they
  do not compete with participants and messages, and a message to self is a single curve;
- colours and shadows are caller options (`WithColors`, `WithShadows`), off by default: in a sequence diagram
  participants get successive palette colours, in graphs the colour says what a node is (a step, a decision, start and
  end, data, input and output); a shadow is an ordinary shape, not a filter.

## Flowchart (`flowchart`, `graph`)

Status: drawn with the layered layout (`internal/layered`).

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header and direction | `flowchart TD`, `graph LR`; directions `TB`, `TD`, `BT`, `LR`, `RL` | FLOW-001 |
| node without a shape | `A` (a rectangle; the identifier is the label) | FLOW-002 |
| shapes | `A[t]`, `A(t)`, `A([t])`, `A[[t]]`, `A[(t)]`, `A((t))`, `A(((t)))`, `A>t]`, `A{t}`, `A{{t}}`, `A[/t/]`, `A[\t\]`, `A[/t\]`, `A[\t/]` | FLOW-002 |
| edges | `-->`, `---`, `-.->`, `-.-`, `==>`, `===`, `--o`, `--x`, `<-->`, `o--o`, `x--x`, `~~~` | FLOW-003 |
| edge label | `A -->\|tekst\| B`, `A -- tekst --> B`, `A == tekst ==> B`, `A -. tekst .-> B` | FLOW-004 |
| edge length | extra characters: `--->`, `----`, `-..->`, `===>`, `~~~~` | FLOW-005 |
| chains and groups | `A --> B --> C`, `A & B --> C & D` | FLOW-006 |
| subgraphs | `subgraph id [tytuł]`, `subgraph id`, `subgraph tytuł ze spacjami` … `end`; `direction` in a subgraph; an edge to a subgraph | FLOW-007 |
| text | quotes for brackets and special characters, `<br/>` | FLOW-008 |
| semicolon | separates statements, may end a line | FLOW-009 |
| drawing | shapes, layout in four directions, wrapped texts | FLOW-012, FLOW-013 |
| styling from the source | `classDef nazwa właściwości`, `class A,B nazwa`, `A:::nazwa`, `style A właściwości`, `linkStyle 0,1 właściwości`, `linkStyle default …`; properties: `fill`, `stroke`, `color`, `stroke-width`, `stroke-dasharray` | FLOW-017 |
| identifiers, Markdown, accessibility | `krok-1.a`; ``"`tekst **pogrubiony** i *pochylony*`"``; `accTitle`, `accDescr` | FLOW-018 |
| subgraph direction | `direction LR` in a subgraph not connected by edges to the rest | FLOW-019 |
| lanes | the comment `%% layout: swimlanes` before or after the header: top-level subgraphs become lanes | FLOW-021 |
| swimlane diagram | the header `swimlane-beta [kierunek]`: lanes from top-level subgraphs; a field in several lanes stays in the first; a subgraph in a lane is part of the lane, without a frame | FLOW-026 |

### Fitting to the page

Layered layout diagrams are fitted to an A4 page by default (`auto` mode): a drawing at least twice as wide as it is
tall is laid out again with narrower fields, in which text wraps more often (LAY-018).
`%% layout: nofit` turns fitting off, `%% layout: fit` turns it on for every drawing that does not fit on the
page; the same is done by the `WithFit` and `WithFitRatio` options and the `-fit` and `-fit-ratio` flags.

A long, simple LR or TB diagram is folded by default into several rows or columns joined by connectors
(LAY-019, ADR 0008): the `return` style (every row in the same direction) or `snake` (every other row reversed),
chosen with the line `%% layout: fold snake` or `fold return`; `%% layout: nofold` turns folding off.

A long chain of fields inside a diagram can be folded into columns (TB) or rows (LR) of its own block (LAY-022). This is
an option, off by default: `%% layout: fold-chains` (with `return` or `snake`), `WithFoldChains`, the `-fold-chains`
flag.

A field with six or more leaves can stand them in two staggered ranks instead of one very long rank
(LAY-025), e.g. an LR flowchart with one field branching out to many fields. Also an option, off by default:
`%% layout: fold-ranks`, `WithFoldRanks`, the `-fold-ranks` flag.

### Outside the profile (error `flowchart-unsupported`)

Shapes of the `@{ shape: … }` syntax that are not drawn under the old syntax (e.g. `doc`, `cloud`, `delay`; the others
work, FLOW-022), `interpolate`, `flowchart-elk`.

### Deliberate differences from Mermaid

- Font Awesome icons in text (`fa:fa-user`) are not drawn: the marker disappears from the label, and a label consisting
  only of an icon gets the icon's name; the drawing contains no third-party icon typefaces;
- text on an edge may also stand without spaces (`A --tekst--> B`);
- a node identifier consists of letters, digits and underscores, with a hyphen or a dot between them;
- style properties other than the five honoured ones (font, rounded corners, transitions) are ignored; an invalid
  value of an honoured property is an error. With a dark fill, the text colour has to be given explicitly (`color`);
- `click` is accepted and ignored: a static drawing has no links;
- the tags `<b>`, `<strong>`, `<i>` and `<em>` in a flowchart label give bold and italic, also when they
  span a line break; other HTML tags are drawn as text (the reference renderer removes them);
- Markdown text may span several source lines, and a line end is a text line break; emphasis must be
  closed on the same line; bold and italic together are drawn as bold
  with an italic attribute, measured with the bold face;
- a node belongs to the subgraph in which it first appeared inside any subgraph; later occurrences
  do not move it;
- an edge between a subgraph and its own contents is an error;
- text in angle brackets (`<znaczniki>`) stays in the drawing verbatim; Mermaid treats it as an HTML tag and removes it.
  This also applies to state diagrams. The only recognised tag is `<br/>`;
- entity codes (`#35;`) are not expanded;
- limits: 500 nodes, 1000 edges, 10 levels of nesting.

### Lanes

Mermaid 12 has a swimlane diagram, `swimlane-beta` (FLOW-026); the `%% layout: swimlanes` directive is a comment, so the
same source is drawn by other tools as an ordinary flowchart with subgraphs. A lane runs through the whole drawing along
the diagram direction (columns for `TB`, rows for `LR`), with a header carrying the title at its start. Ranks are shared
by all lanes: the position of a step along a lane tells its order in the whole process. A handover between lanes (an
edge without a label) stays in one rank and runs across, from the side of one field to the side of the other, if it does
not pass through a field of another lane (LAY-027); other edges change lanes once, at their start. A field without edges
stands after the previous field of its lane (FLOW-029). In horizontal lanes the title is rotated, and in colour mode
lanes get successive palette tones (FLOW-028). A field that is the only field of its lane in its row stands in the
middle of the lane, so the steps of one role form a column. Lanes exist only in flowcharts: a composite state is not a
role, so state diagrams do not have them. Lanes cannot contain nested subgraphs or edges to the lane itself (an error),
and `direction` in a subgraph has no effect.

### Drawing limitations (shared with state diagrams)

- `direction` in a subgraph or in a composite state changes the direction of the interior only when no edge connects the
  interior with the rest of the diagram (Mermaid decides this the same way); otherwise the interior takes the
  surrounding direction;
- an edge leaving a subgraph runs from its own point on the frame; the points on the frame are ordered by the other
  ends of the edges outside, as long as this does not add crossings inside the subgraph;
- concurrent regions are separate fields with a slightly darkened background and a dashed outline, without a title;
- a level made up only of unconnected frames (regions, separate subgraphs) is packed freely: lower frames stand
  one below the other next to a taller one, in source order; the measure is the scale at which the whole fits the page;
- parts of a diagram (and of the interior of a subgraph or composite state) not connected by edges are arranged in rows
  so that the whole has proportions close to the page (3:4); within a row they stand side by side, in source order;
- an edge pointing at a subgraph ends at its frame; an edge to a node inside a subgraph passes through a point on the
  frame and is routed inside between nodes (FLOW-014); it may pass under the frame title, which then gets a backing;
- an invisible edge (`~~~`) affects only placement;
- an edge with a label is one rank longer: the label takes up a rank like a field, next to the fields of other branches,
  while edges without labels stay short. Multiple edges between the same pair of fields are also one rank longer;
- each extra line character (`--->`) lengthens the edge by one rank, also when the target has no further connections;
  in that case the reference renderer places the targets in one line;
- edges are drawn orthogonally, with rounded corners; cross segments in one gap between
  ranks get separate tracks, and the gap grows with their number. The `WithCurvedEdges` option (`-curved` in the program)
  restores gentle curves. An orthogonal edge turns once, at the end from which several edges fan out; its
  label stands on the axis of the other end, and a field with several branches stands in the middle of them;
- a decision diamond with one branch running straight down: the other branches leave from the side vertices and run
  sideways at the height of the diamond's centre, as in a classic flowchart (the main flow along the axis, the branch to
  the side). When no branch runs straight or something is in the way, the branches leave from the sides near the bottom
  vertex. The rule does not apply to backward (upward) edges or to curves (`WithCurvedEdges`);
- in lanes, the order of the fields of one lane is adjusted after grouping so as not to add crossings; the intermediate
  points of an edge whose two ends each have several edges stand on one axis near the lower end;
- an entity relationship diagram is laid out for the page proportions: a rank that is too wide hands fields (entities
  with one relationship first) to the rank below, so a large model grows downwards instead of sideways; a small diagram
  stays unchanged;
- empty columns are closed: a strip in which nothing stands in any rank shrinks to the normal gap,
  and the edges that crossed it become shorter;
- where orthogonal edges cross, the edge written later in the source jumps over the earlier one with a semicircular
  bridge, as long as there is room for it alongside; this also applies to edges from different nesting levels of
  subgraphs;
- an edge's attachment point moves onto the axis of the edge's next point if that point lies within the field's side:
  the edge then runs straight, without a small jog;
- a circle or cross edge end stands next to the field, and the line ends at it, not at the field;
- fields joined in a simple sequence (one outgoing and one incoming edge) have the same size across
  the diagram direction, and the fields of one rank the same size along it, as long as they differ by at most 40%;
  the widest field with one line of text is wrapped onto two if it then fits within the width of its neighbour;
- a field with one neighbour is placed on that neighbour's axis first; a field placed between several neighbours
  gives way to it;
- fields are flat, at least 140 px wide with a 16 px font; edges leave a diamond and a circle separately,
  from the sides near the vertex;
- a state diagram note stands next to its state on the indicated side of the ordering axis: in a vertical layout on the
  left or on the right, in a horizontal layout above or below the state respectively;
- labels wider than 200 px (150 px in a diamond and a circle) are wrapped at word boundaries.

## State diagram (`stateDiagram-v2`, `stateDiagram`)

Status: drawn with the same layered layout.

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header | `stateDiagram-v2`, `stateDiagram` | STATE-001 |
| transition | `A --> B`, `A --> B : etykieta`; `[*]` as the initial and final state | STATE-002 |
| state description | `state "Opis" as Id`, `Id : opis` (subsequent lines append) | STATE-003 |
| state without a description | used in a transition, or the identifier alone on a line | STATE-004 |
| composite state | `state Id { … }`, `state "Opis" as Id { … }`, nested; a transition to a composite state | STATE-005 |
| choice, fork, join | `state Id <<choice>>`, `<<fork>>`, `<<join>>` | STATE-006 |
| note | `note right of Id : tekst`, `note left of Id` … `end note` | STATE-007 |
| drawing | state shapes, layout, texts | STATE-010, STATE-011 |
| direction | `direction LR` at the diagram level or in a composite state | STATE-008 |
| concurrent regions | `--` inside a composite state | STATE-012 |
| styling from the source | `classDef`, `class A nazwa`, `A:::nazwa`, `style` | STATE-013 |
| accessibility | `accTitle`, `accDescr` | STATE-014 |

### Outside the profile (error `state-unsupported`)

`scale`, `hide`, `state Id` without a description, stereotype or braces (a form not described in the documentation; the
reference renderer does not draw it).

### Deliberate differences from Mermaid

- `subgraph` from flowcharts is an error with a hint (Mermaid then draws the words "subgraph", the group
  name and "end" as three separate states);
- a state identifier consists of letters, digits and underscores; a hyphen is an error, as in Mermaid;
- a composite state has its title in a separate bar at the top of the frame, with a light background and a line beneath,
  and the frame has more rounded corners; at the bottom the frame is closed by a narrow bar (4 px) in the colour of the
  title bar, without text, so that a large state frame can be recognised from below too; a flowchart subgraph keeps its
  title by the left edge;
- inside a composite state (and a region), straight arrows without a label are shorter than at the top level
  (rank gap 26 instead of 44 px); the gap at a label and at an edge with a turn stays unchanged;
- a transition between a composite state and its own contents is an error.

## Gantt chart (`gantt`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, title, sections | `gantt`, `title Tekst`, `section Nazwa` | GANTT-001 |
| task | `Nazwa : [znaczniki,] [id,] [początek,] koniec` | GANTT-002 |
| start | a date in the `dateFormat` notation, `after id1 id2`, or nothing (after the previous task) | GANTT-002 |
| end | a date, a duration (`3d`, `2w`, `12h`, `1.5d`, `6M`, `1y`), `until id` | GANTT-002 |
| tags | `done`, `active`, `crit`, `milestone` | GANTT-003 |
| date notation | `dateFormat` from the tokens `YYYY`, `YY`, `MM`, `M`, `DD`, `D`, `HH`, `mm`, `ss` | GANTT-002 |
| excluded days | `excludes weekends`, day names, `YYYY-MM-DD` dates; `weekend friday`; `inclusiveEndDates` | GANTT-004 |
| axis ticks | `axisFormat` (strftime: `%Y %y %m %d %e %H %M %S %j %b %B %a %A`), `tickInterval 1day`/`week`/`month`/`year` | GANTT-006 |
| accessibility | `accTitle`, `accDescr` | GANTT-001 |

### Outside the profile (error `gantt-unsupported`)

`includes`, `displayMode compact`, vertical markers `vert`.

### Deliberate differences from Mermaid

- the "today" marker is never drawn (`todayMarker` is accepted and ignored): the same text must always give
  the same drawing;
- with `excludes`, a duration counts working time: excluded days do not count, also when a task starts on such a
  day; an end given as a date is not moved;
- `tickInterval` is honoured as long as the tick labels do not touch; otherwise the interval grows;
- month and day names in `axisFormat` are in English; `topAxis` and `click` are accepted and ignored;
- a task name ends at the first colon;
- limits: 500 tasks, years 1900–2200, a span of up to 100 years.

## Entity relationship diagram (`erDiagram`)

Status: drawn with the layered layout; an entity is a field with a header and a table of attributes.

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, direction | `erDiagram`, `direction LR` | ER-001 |
| entity | `NAZWA`, `"Nazwa ze spacją"`, `NAZWA["Opis"]`, `NAZWA { … }` | ER-002 |
| attribute | `typ nazwa`, `typ nazwa PK`, `typ nazwa PK, FK "komentarz"` | ER-002 |
| relationship | `A \|\|--o{ B : etykieta`; cardinalities `\|o`, `\|\|`, `}o`, `}\|` and their mirror forms; line `--` or `..` | ER-003 |
| styling from the source | `classDef`, `class`, `NAZWA:::klasa`, `style` | ER-004 |
| accessibility | `accTitle`, `accDescr` | ER-001 |

### Outside the profile

Relationships written in words (`only one to zero or more`; error `er-unsupported`), attributes with types containing a
space.

### Deliberate differences from Mermaid

- an unquoted entity name consists of letters, digits, underscores and hyphens;
- a relationship label is optional;
- a limit of 100 attributes per entity; graph limits as in flowcharts.


## Pie chart (`pie`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header | `pie`, `pie showData`, `pie title Tekst`, `pie showData title Tekst` | PIE-001 |
| title and values in the legend | `title Tekst`, `showData` on a separate line | PIE-001, PIE-004 |
| slice | `"etykieta" : 42.5` | PIE-002 |
| accessibility | `accTitle`, `accDescr` | PIE-001 |

### Outside the profile

Configuration in the `%%{init}%%` and `---` blocks (text position, outline width), themes.

### Deliberate differences from Mermaid

- slices are in source order, clockwise from twelve o'clock;
- the default fills are shades of grey of alternating lightness, so the chart can be read in black-and-white
  print; `WithColors` gives palette colours;
- the percentage stands on a slice only when the slice covers at least 4% of the circle; a smaller one cannot be
  labelled without overlapping its neighbours (the reference renderer labels every one);
- the share is rounded to a whole number, and below 10% to one decimal place;
- a slice label cannot be empty or repeated, and a negative value is an error; a chart without a slice
  with a value above zero is an error; the limit is 24 slices.

## Class diagram (`classDiagram`)

Drawn with the same layered layout as flowcharts; the same edge rules and the same
drawing limitations apply.

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header | `classDiagram`, `classDiagram-v2` | CLS-001 |
| class | `class Nazwa`, `class Nazwa["Etykieta"]`, `class Lista~T~` | CLS-001 |
| members | `class Nazwa { … }` one per line; `Nazwa : +pole`; `Nazwa : +metoda(arg) Typ` | CLS-002 |
| classifiers | `metoda()*` (abstract, italic), `metoda()$` and `pole$` (static, underlined) | CLS-003 |
| annotation | `<<interface>>` in the class block; `<<interface>> Nazwa` | CLS-003 |
| relationships | `<\|--`, `*--`, `o--`, `-->`, `--`, `..>`, `..\|>`, `..` and the reversed forms | CLS-004 |
| label and cardinalities | `A "1" --> "0..*" B : etykieta` | CLS-005 |
| direction, note | `direction LR`; `note for Nazwa "tekst"` | CLS-006 |
| styling | `classDef`, `class Nazwa:::styl`, `cssClass "A,B" styl`, `style Nazwa …` | CLS-007 |
| accessibility, links | `accTitle`, `accDescr`; `click`, `link`, `callback` (ignored) | CLS-006 |
| namespace | `namespace Nazwa { class A … }` | CLS-009 |
| one-line block, note | `class A { <<interface>> }`; `note "tekst"` | CLS-009 |
| "lollipop" interface | `Interfejs ()-- Klasa`, `Klasa --() Interfejs` | CLS-009 |

### Outside the profile (error `class-unsupported`)

Nested namespaces, class names in backticks, the `hideEmptyMembersBox` configuration.

### Deliberate differences from Mermaid

- a note spread over several lines shows the whole text (Mermaid draws only the first line);
- a class without members is a plain field with its name; the reference renderer always draws three compartments;
- a class with members has two compartments; an empty one remains as a narrow bar;
- directed association and dependency have an open arrowhead (two strokes), as in UML, not a filled one;
- cardinalities stand beside the line right next to the field; an edge with cardinalities at both ends is one rank longer,
  so the labels do not touch;
- a class identifier is the name without the type parameter: `Lista~T~` and `Lista` are the same class;
- a colour given to a class by a style covers the whole field, compartment lines included;
- a "lollipop" interface is an ordinary relationship with a circle at the end, and the interface itself is a class field
  (in Mermaid just a circle with a name).

## Timeline (`timeline`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, title | `timeline`; `title Tekst` | TML-001 |
| period with events | `2024 : zdarzenie : zdarzenie`; a following line `: zdarzenie` appends events | TML-002 |
| section | `section Tytuł` | TML-003 |
| list layout | the comment `%% layout: list` | TML-006 |
| accessibility | `accTitle`, `accDescr` | TML-001 |

### Outside the profile

Configuration in the `%%{init}%%` block (`disableMulticolor`, themes).

### Layouts

The default layout is as in Mermaid: periods in a row from left to right, below them an axis with an arrow, below the axis
the events of each period one below the other, and a section header above its periods. A longer axis then does not fit
the page width and is scaled down.

The list layout (`%% layout: list`) runs the axis from top to bottom: periods stand on the left of the axis and a
period's events on the right, one below the other in source order, joined by a dashed line. The drawing grows downwards
and fits the page for any number of periods. The directive is a comment, so the same source is drawn in the default
layout by other tools.

### Deliberate differences from Mermaid

- `<br/>` in period and event text breaks the line (the reference renderer draws the tag verbatim);
- a direction given after the header (`timeline LR`, `timeline TD`) is accepted and ignored;
- by default the drawing is neutral; `WithColors` gives a palette tone to each section, or to each period when there are
  no sections;
- a colon always separates events, so text with a colon (e.g. the time `10:30`) has to be written differently;
  the same holds in Mermaid.

## User journey (`journey`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, title | `journey`; `title Tekst` | JRN-001 |
| step | `Nazwa kroku: 4: Uczestnik, Uczestnik`; participants optional | JRN-002 |
| section | `section Tytuł` | JRN-003 |
| list layout | the comment `%% layout: list` | JRN-005 |
| accessibility | `accTitle`, `accDescr` | JRN-001 |

### Layouts

The default layout is as in Mermaid: steps in a row under the section headers, under each step a face at the height of
its score, an axis with an arrow at the bottom, a legend of participants on the left, and their markers on the steps.

The list layout (`%% layout: list`) places the steps from top to bottom: the name, a score marker on a scale from 1 to 5,
and the participants listed by the step; a line joins the scores of successive steps. It fits the page with many steps,
and the score is readable in it without colour (position on the scale; score marker 1–2 filled, 3 grey, 4–5 hollow).

### Deliberate differences from Mermaid

- the face has three expressions: a smile for scores 4 and 5, a straight mouth for 3, sadness for 1 and 2;
- by default the drawing is neutral and participants differ in shade of grey; `WithColors` gives palette colours
  to sections, participants and faces;
- a score outside the range 1–5, or a non-integer one, is an error.

## Mind map (`mindmap`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, hierarchy | `mindmap`; successive lines, indentation determines the parent | MND-001 |
| branch | plain text; `"tekst w cudzysłowach"`; `<br/>` breaks the line | MND-002 |
| shapes | `id[kwadrat]`, `id(zaokrąglony)`, `id((koło))`, `id{{sześciokąt}}`, `id))wybuch((`, `id)chmura(` | MND-004 |
| Markdown text | ``"`**pogrubiony** i *pochylony*`"`` | MND-004 |
| icons, classes | `::icon(…)`, `:::klasa` (ignored) | MND-002 |
| list layout | the comment `%% layout: list` | MND-006 |

### Layouts

By default the root stands in the middle, with the main branches on both sides of it, as in a classic mind map. The list
layout (`%% layout: list`) places all branches to the right of the root and reads like an outline. The directive is a
comment, so the same source is drawn in the default layout by other tools.

### Deliberate differences from Mermaid

- both layouts are a tree with branches of the same depth in one column and right-angled connections; Mermaid
  spreads branches freely around the root and joins them with curves. The tree is reproducible and fits the page;
- "bang" is drawn as a hexagon and "cloud" as a field with semicircular ends: the drawing has only simple shapes;
- a circle with longer text becomes an ellipse, so the field does not grow out of proportion;
- a branch without a shape is a rounded field (in Mermaid, underlined text), and a root without a shape is a field
  with semicircular ends;
- quotes around text are removed (the reference renderer draws them verbatim);
- a tab in indentation counts as four spaces;
- by default the drawing is neutral; `WithColors` gives each main branch a palette tone;
- `accTitle` and `accDescr` are not part of the mind map syntax: such a line would be a branch.

## Bar and line chart (`xychart-beta`, `xychart`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, title | `xychart-beta`; `title "Tekst"` | XY-001 |
| horizontal axis | `x-axis [sty, lut, mar]`; `x-axis "Tytuł" [a, b]`; `x-axis Tytuł 0 --> 10` | XY-002 |
| vertical axis | `y-axis "Tytuł" 0 --> 40`; `y-axis Tytuł`; `y-axis 0 --> 40` | XY-002 |
| series | `bar [1, 2, 3]`; `line [1, 2, 3]`; with a name: `bar "Plan" [1, 2, 3]` | XY-002, XY-004 |
| accessibility | `accTitle`, `accDescr` | XY-001 |

### Outside the profile

Configuration in the `%%{init}%%` and `---` blocks (sizes, `showDataLabel`).

Horizontal chart (`xychart-beta horizontal`, XY-006): categories from top to bottom, values horizontal; the category
axis title is rotated along the axis.

### Deliberate differences from Mermaid

- the vertical axis title stands horizontally above the axis, not rotated along it;
- without a given range, the vertical axis of a bar chart includes zero, because a bar is measured from zero; the ticks
  have round values;
- several bar series stand side by side within a category (in Mermaid they overlap);
- series differ without colour: bars by lightness, lines by dash pattern; `WithColors` gives palette colours;
- a series name before the list of values gives a legend below the chart;
- when category labels do not fit, only every n-th one is kept, instead of overlapping;
- a series must have as many values as there are categories; a different number is an error.
- the title and categories are also accepted without quotes when they contain non-ASCII letters or several words; Mermaid
  then requires quotes, so it is worth writing them in portable source;
- the vertical axis ticks use a different step than in Mermaid, so the numbers on the axis differ.

## Quadrant chart (`quadrantChart`)

### Supported

| Element | Syntax | Requirement |
|---|---|---|
| header, title | `quadrantChart`; `title Tekst` | QDR-001 |
| axes | `x-axis Lewy --> Prawy`; `y-axis Dolny --> Górny`; the second label optional | QDR-001 |
| quadrants | `quadrant-1 Tekst` … `quadrant-4 Tekst` | QDR-001 |
| point | `Nazwa: [0.3, 0.6]` | QDR-002 |
| point styles | `Nazwa:::klasa: [x, y]`, `radius: 12, color: #f00, stroke-color: #000, stroke-width: 2px` after the bracket, `classDef` | QDR-002 |
| accessibility | `accTitle`, `accDescr` | QDR-001 |

### Deliberate differences from Mermaid

- a point's name looks for a place: below the dot, and when it would overlap a quadrant title, another dot or name
  there, above the dot or beside it; a quadrant title that a dot lies on starts right below it (in Mermaid the labels
  overlap);
- quadrants that share a side differ in lightness; `WithColors` gives them palette tones;
- a coordinate outside the range 0–1 is an error.

## Kanban board (`kanban`)

| Element | Syntax | Requirement |
|---|---|---|
| column | a line at the shallowest indentation: `Tekst` or `id[Tekst]` | KBN-001 |
| card | a line indented deeper: `[Tekst]`, `id[Tekst]` or plain text | KBN-002 |
| card data | `@{ ticket: NC-17, assigned: 'A. Nowak', priority: 'High' }` | KBN-002 |
| accessibility | `accTitle`, `accDescr` | KBN-001 |

Deliberate differences from Mermaid: the priority (`Very High`, `High`, `Low`, `Very Low`) is shown by the thickness of
a bar at the left edge of the card, not by colour; the card data stands on one line below the text; the ticket link
(`ticketBaseUrl`) is not drawn; by default the drawing is neutral, `WithColors` gives the columns palette tones.

## Ishikawa diagram (`ishikawa-beta`, `ishikawa`)

| Element | Syntax | Requirement |
|---|---|---|
| problem | the first line after the header | ISH-001 |
| category | a line at the indentation of the first cause | ISH-002 |
| cause | a line indented deeper than its category or parent cause (up to eight levels) | ISH-002 |
| accessibility | `accTitle`, `accDescr` | ISH-001 |

Deliberate differences from Mermaid: an own look (a spine with an arrowhead to the problem field on the right,
categories alternately above the spine and below it in fields at the ends of slanted bones, causes on horizontal
strokes); more than eight levels of causes under a category is an error.

## Tree view (`treeView-beta`, `treeView`)

| Element | Syntax | Requirement |
|---|---|---|
| entry | a name on its own line, plain or `"quoted"`; a trailing `/` marks a directory | TREE-003 |
| nesting | indentation (a tab counts as four spaces) or box-drawing characters `├──`, `└──`, `│` and the heavy `┣━━`, `┗━━`, `┃`; the output of `tree` (a `.` root with branches from column zero) is read as is | TREE-002 |
| highlight | `:::highlight` marks the whole entry, its name and description; other `:::class` names are accepted and do not change the drawing | TREE-003 |
| icon | `icon(folder)`, `icon(file)`, `icon(pack:name)`, `icon(none)` | TREE-003, TREE-004 |
| description | `## text` at the end of the line; grey italics, green with `WithColors` | TREE-003 |
| comment | `%%` | TREE-001 |
| accessibility | `accTitle`, `accDescr` | TREE-001 |

Deliberate differences from Mermaid: there is no `/` root above the entries, and a directory keeps the slash it was
written with; an entry with children is a directory even without a slash; descriptions stand in one column after
the longest name that has one; icon packs are not bundled, so `icon(pack:name)` draws this library's own file or
folder icon by the kind of entry; the `treeView` configuration (`showIcons`, `rowIndent`, icon maps) and its theme
variables are outside the profile. At most 2000 entries and 32 levels.

## Venn diagram (`venn-beta`, `venn`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Tekst`, `title "Tekst"` | VENN-001 |
| set | `set A`, `set A["label"]`, `set "Long name"`, optional size `:20` | VENN-002 |
| overlap | `union A,B["label"]`, `union A,B,C`, optional size `:3`; the sets must be declared first | VENN-002 |
| item | `text A1["label"]` on an indented line under a set or overlap | VENN-002 |
| style | `style A fill:#f96`, `style A,B color:#333`, `style A1 color:red`: `fill`, `fill-opacity`, `stroke`, `stroke-width`, `color` | VENN-003 |
| accessibility | `accTitle`, `accDescr` | VENN-001 |

Deliberate differences from Mermaid: an own look (circles with a light translucent fill, so overlaps are darker;
neutral by default); at most three sets, so every overlap can be drawn as a region of circles; a set without any
overlap stands apart to the right; circle areas follow the sizes and an overlap's size sets the distance of its
circles, without a numerical fit of all overlaps at once; a style on an overlap fills the whole common area of its
circles; every label block fits inside its region and is centred in the room available there:
when text does not fit, all circles grow (up to 1.3 times), then all text shrinks (down to 0.8), and a label that
still does not fit stands outside the circles with a leader line to a dot in its region.

## Cynefin map (`cynefin-beta`, `cynefin`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Tekst`, `title "Tekst"` | CYN-001 |
| domain | `complex`, `complicated`, `clear`, `chaotic`, `confusion` on its own line; may be repeated | CYN-002 |
| item | `"Tekst"` on the lines after a domain name | CYN-002 |
| transition | `complex --> complicated : "label"`, label optional; a transition to the same domain is ignored | CYN-003 |
| accessibility | `accTitle`, `accDescr` | CYN-001 |

Deliberate differences from Mermaid: an own look (a framed board with gently bent boundaries, the cliff between
Clear and Chaotic drawn as a heavier lower half of the vertical boundary, domain text at the outer edges so the middle
stays free for transitions, items in framed boxes; neutral by default, `WithColors` tints the domains); transitions are
arcs bent away from the centre with the label on their outer side; the `cynefin` configuration (`width`, `height`,
`showDomainDescriptions`, `boundaryAmplitude`, `seed`) and its theme variables are outside the profile; an item must be
quoted; at most 120 items.

## Use case diagram (`usecase-beta`, `usecase`)

| Element | Syntax | Requirement |
|---|---|---|
| direction | `direction TB`, `TD`, `BT`, `LR`, `RL` (default TB) | UC-001 |
| actor | `actor Id`, `actor Id("label")`, metadata `@{ type: hollow }`, `@{ business: true }`, `@{ icon: "fa:user" }`, stereotype `<<Name>>`, `:::class` | UC-002 |
| use case | `Id("label")` (ellipse), `Id[label]` (rectangle), `"Name"`, `Id`; an undeclared relationship end | UC-002 |
| label | plain, `"quoted"`, Markdown string in backticks, codes `#quot;` `#39;` `#40;` `#41;` `#91;` `#93;` `#96;` | UC-002 |
| system boundary | `systemBoundary Id["title"]` or `systemBoundary "title"` … `end`, one level; `@{ type: package }` | UC-003 |
| association | `-->`, `--->`, `<--`, `--`, `--o`, `o--`, `--x`, `x--`; `A -- "label" --> B` | UC-004 |
| include, extend | `A ..> : include B`, `A ..> : extend B` | UC-004 |
| generalisation | `A --|> B` (two actors or two use cases) | UC-004 |
| edge id | `A id@--> B`; metadata `id@{ animation: fast }` is accepted | UC-004 |
| note | `note for Id "text"` | UC-004 |
| JSON table | `json Id@{ … }` | UC-005 |
| style | `classDef`, `class`, `style`, `:::class` on actors, use cases, boundaries, tables and edges with an id | UC-005 |
| accessibility | `accTitle`, `accDescr`, `accDescr { … }` | UC-001 |

Deliberate differences from Mermaid: the layered layout of flowcharts instead of ELK; every actor is the same stick
figure (the `hollow`, `awesome` and icon variants and the business slash are not drawn); a `package` boundary is
drawn like a plain one; edge animation is not drawn; identifiers may contain any letters; a class must be defined
with `classDef` before the end of the source; the `usecase` configuration and `colorScheme` are outside the profile.

## ZenUML sequence diagram (`zenuml`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Text` on the first line after the header | ZEN-001 |
| participant | implicit on first use; `Name`, `A as Alice`, `@Actor Name`, `@Database Name` and other annotators | ZEN-002 |
| asynchronous message | `A->B: text`; `@return` or `@reply` before it makes a reply | ZEN-003 |
| comment | `// text` before a message (drawn as a note over it) | ZEN-003 |
| synchronous call | `B.method(args)`, `A->B.method()`, block `{ … }` with nested calls | ZEN-004 |
| reply | `return value`, `result = B.method()`, `Type result = B.method()` | ZEN-004 |
| creation | `new A`, `new A(args)`, `a = new A()` | ZEN-004 |
| loop | `while(…)`, `for(…)`, `forEach(…)`, `foreach(…)`, `loop` | ZEN-005 |
| alternative | `if(…) { } else if(…) { } else { }` | ZEN-005 |
| other blocks | `opt { }`, `par { }`, `try { } catch { } finally { }` | ZEN-005 |
| accessibility | `accTitle`, `accDescr` | ZEN-001 |

Drawn with the layout of `sequenceDiagram`. Deliberate differences from ZenUML: a call at the top level comes from an
unnamed stick figure on the left; annotators other than `@Actor` are shown as a stereotype above the name instead of
an icon; a comment becomes a note over the message; Markdown in comments is shown as plain text; `par` has one
section; a reply to a variable after `new` comes from the created participant.

## Wardley map (`wardley-beta`, `wardley`)

| Element | Syntax | Requirement |
|---|---|---|
| title, size | `title Text`, `size [900, 560]` | WARD-001 |
| component | `component Name [visibility, evolution]`, quoted names, `label [dx, dy]`, `(build)`, `(buy)`, `(outsource)`, `(market)`, `(inertia)` | WARD-002 |
| anchor | `anchor Name [visibility, evolution]` | WARD-002 |
| pipeline | `pipeline Name { component "part" [evolution] … }` | WARD-002 |
| link | `A -> B`, `A --> B`, `A -> B; label`, `A -.-> B`, `A +> B`, `A +< B`, `A +<> B`, `A +'label'> B` | WARD-003 |
| evolution | `evolve Name 0.8`, trend `Name -.- (x, y)`, stages `evolution A -> B@0.4 -> C / D` | WARD-004 |
| notes and forces | `note "text" [v, e]`, `annotations [v, e]`, `annotation 1,[v, e] "text"`, `accelerator "name" [v, e]`, `deaccelerator "name" [v, e]` | WARD-004 |
| comment | `%%`, also at the end of a line | WARD-001 |
| accessibility | `accTitle`, `accDescr` | WARD-001 |

Deliberate differences from Mermaid: an own look (neutral; `WithColors` draws evolution red and forces tinted);
default stages Genesis, Custom built, Product (+ rental), Commodity (+ utility) with boundaries at 0.17, 0.40 and 0.70;
names may contain any letters; the positions of `annotations` and `annotation` are read as [visibility, evolution]
like all other positions; overlapping labels are not moved.

## Event model (`eventmodeling`)

| Element | Syntax | Requirement |
|---|---|---|
| time frame | `tf 01 cmd Name`, `timeframe 01 command Name`; types `ui`, `pcr`/`processor`, `cmd`/`command`, `rmo`/`readmodel`, `evt`/`event` | EVM-002 |
| namespace | `Namespace.Name` | EVM-002 |
| data | inline `{ key: value, … }`, reference `[[Block]]` to `data Block { … }`, optional type in backticks | EVM-003 |
| reset frame | `rf 05 …`, `resetframe 05 …` | EVM-003 |
| explicit relation | `->> 02`, several in a row | EVM-003 |
| accessibility | `accTitle`, `accDescr` | EVM-001 |

Deliberate differences from Mermaid: an own look (lanes with a header column, one column per frame, elements with
their type, name and data; the type is also told by the shape; neutral by default, `WithColors` gives the usual Event
Modeling colours); relations are straight arrows; data types (`json`, `figma` and others) are accepted and not
interpreted, as in Mermaid.

## Agent flow (`agentflow`, `agentflow-beta`)

| Element | Syntax | Requirement |
|---|---|---|
| header | `agentflow`, `agentflow-beta`, optional `TB`, `TD`, `BT`, `LR`, `RL` | AGF-001 |
| node | `id["label"]@{ shape: task }`; aliases `task`, `tool`, `input`, `decision`, `refdoc`, `action`, or a flowchart shape name; `label`; other keys accepted; metadata on one or several lines | AGF-002 |
| edge | `-->`, `-.-`, `--x`, `a -- yes --> b`, chains | AGF-003 |
| container | `flow id["label"] … end`, nested; `global … end` | AGF-004 |
| collapse | `id@{ view: "collapsed" }` | AGF-004 |
| connector | `connector id["label"]@{ … }`; `connectorRef: id` on a tool | AGF-004 |

Drawn as a flowchart after translating the source line by line. Deliberate differences from Mermaid: the layered
layout of flowcharts instead of ELK and the neutral look instead of `redux-color`; `refdoc` is a rectangle with a
dashed outline; a connector is a pill joined to the tools that refer to it by a dotted line; a collapsed container is
a subroutine box; the per-container `algorithm` and edge animation are accepted and not drawn; the semantic model and
diagnostics are not produced.

## Railroad diagram (`railroad-ebnf-beta`, `railroad-abnf-beta`, `railroad-peg-beta`, `railroad-beta`)

| Element | Syntax | Requirement |
|---|---|---|
| header and rules | one of the four headers (also without `-beta`), `title`, rules ending in `;` on one or several lines | RAIL-001 |
| EBNF | `=` or `::=`, `"x"` `'x'`, names, sequence by spaces or commas, `|`, `A?` `[A]`, `A*` `{A}`, `A+`, `( )`, `A - B`, `? text ?`, `/* */`, `(* *)` | RAIL-002 |
| ABNF | `=`, `/`, `*A`, `1*A`, `n*mA`, `nA`, `[A]`, `( )`, `%x41`, `%x30-39`; `;` ends the rule and starts a comment | RAIL-003 |
| PEG | `<-`, `/`, `?` `*` `+`, `&A`, `!A`, `.`, `[a-z]`, `#` comments | RAIL-003 |
| constructors | `terminal`, `nonterminal`, `special`, `sequence`, `choice`, `optional`, `zeroOrMore`, `oneOrMore` | RAIL-004 |
| accessibility | `accTitle`, `accDescr` | RAIL-001 |

Deliberate differences from Mermaid: an own look (start dot and end ring, terminals in rounded boxes, non-terminals in
rectangles, special sequences, exceptions, predicates and "any character" in italic hexagons; a repetition loop has an
arrow and, for ABNF counts, a label such as "2–4×"); the first branch of a choice stays on the track; the `railroad`
configuration block is outside the profile.

## Packet diagram (`packet-beta`, `packet`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Tekst` | PKT-001 |
| field | `0-15: "nazwa"`, `16: "nazwa"`, `+8: "nazwa"` | PKT-002 |
| accessibility | `accTitle`, `accDescr` | PKT-001 |

Deliberate differences from Mermaid: a row always has 32 bits (the `bitsPerRow` configuration is outside the profile); a
name that does not fit in its field is drawn in a smaller font; the title stands above the diagram; a gap between fields
is an error.

## Radar chart (`radar-beta`, `radar`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Tekst` | RDR-001 |
| axes | `axis a["Podpis"], b["Podpis"]` | RDR-001 |
| series | `curve s["Podpis"]{1, 2, 3}`; `curve s{a: 1, b: 2, c: 3}` | RDR-002 |
| range and grid | `max 5`, `min 0`, `ticks 5`, `graticule circle` or `polygon`, `showLegend false` | RDR-002 |
| accessibility | `accTitle`, `accDescr` | RDR-001 |

Deliberate differences from Mermaid: series differ by line dash pattern, so the chart reads without colour (`WithColors`
gives palette colours); the legend stands on the right; a series must have a value for every axis; a value outside the
range is drawn at its edge.

## Treemap (`treemap-beta`, `treemap`)

| Element | Syntax | Requirement |
|---|---|---|
| title | `title Tekst` | TRM-001 |
| section | `"Nazwa"`, children indented deeper | TRM-001 |
| leaf | `"Nazwa": 12.5` | TRM-001 |
| styles | `:::klasa`, `classDef` (ignored) | TRM-001 |
| accessibility | `accTitle`, `accDescr` | TRM-001 |

Deliberate differences from Mermaid: a leaf whose name does not fit in its field even in a smaller font gets a number,
and the name with its value goes into a list below the map, so no data disappears; the value is written as in the source,
without formatting; the map has a fixed size of 620 × 400 px; by default the drawing is neutral, `WithColors` gives a
palette tone to each top-level branch.

## Requirement diagram (`requirementDiagram`)

Drawn with the same layered layout as flowcharts and class diagrams.

| Element | Syntax | Requirement |
|---|---|---|
| requirement | `requirement nazwa { id: 1  text: …  risk: high  verifymethod: test }` and five specific kinds | RQD-001, RQD-002 |
| element | `element nazwa { type: …  docref: … }` | RQD-001, RQD-002 |
| relationship | `a - satisfies -> b`; `b <- satisfies - a` | RQD-003 |
| direction, styling | `direction LR`; `classDef`, `class`, `nazwa:::styl`, `style` | RQD-003 |
| accessibility | `accTitle`, `accDescr` | RQD-001 |

Deliberate differences from Mermaid: long requirement text is wrapped into lines of about 44 characters; the `contains`
relationship has a circle at the containing element (in Mermaid a circle with a cross); a name used in a relationship
must be defined by a block; a field value is also accepted without quotes when it contains a hyphen or a dot (Mermaid
then requires quotes).

## Sankey diagram (`sankey-beta`, `sankey`)

| Element | Syntax | Requirement |
|---|---|---|
| flow | `źródło,cel,wartość`; `"nazwa, z przecinkiem",cel,12.5` | SNK-001 |
| name wrapping | `%% layout: wrap` (node names wrapped at words) | SNK-006 |

Nodes are positioned vertically relative to their neighbours, and a flow across several columns avoids the nodes of the
intermediate columns, so the bands run as straight as possible and do not cross (SNK-005).

Deliberate differences from Mermaid: each node shows its value; bands are grey and semi-transparent by default, and
`WithColors` gives them the colour of the source node; the chart has a fixed width of 620 px and a height that grows
with the number of nodes in a column; node names may contain non-ASCII letters (Mermaid rejects characters outside ASCII
in this type, even in quotes); a cycle of flows is an error; `accTitle` and `accDescr` are not part of the syntax (the
line would be a flow).

## C4 diagrams (`C4Context`, `C4Container`, `C4Component`, `C4Dynamic`, `C4Deployment`)

Drawn with the same layered layout as flowcharts.

| Element | Syntax | Requirement |
|---|---|---|
| person, system | `Person(alias, "Etykieta", "Opis")`, `System(…)`, with the suffixes `Db`, `Queue`, `_Ext` | C4D-001 |
| container, component | `Container(alias, "Etykieta", "Technologia", "Opis")`, `Component(…)`, with the same suffixes | C4D-001 |
| relationship | `Rel(a, b, "Etykieta", "Technologia")`, `BiRel`, `Rel_Back`, `Rel_U`/`D`/`L`/`R`, `RelIndex(1, a, b, "…")` | C4D-002 |
| boundary | `Enterprise_Boundary(alias, "Etykieta") { … }`, `System_Boundary`, `Container_Boundary`, `Boundary(alias, "Etykieta", "rodzaj")` | C4D-003 |
| deployment node | `Deployment_Node(alias, "Etykieta", "Typ") { … }`, `Node(…)` | C4D-003 |
| title, accessibility | `title Tekst`; `accTitle`, `accDescr` | C4D-003 |
| styling, layout | `UpdateElementStyle(…)`, `UpdateRelStyle(…)`, `UpdateLayoutConfig(…)`, `$klucz="…"` arguments (ignored) | C4D-003 |

Deliberate differences from Mermaid: the layout is layered, determined by the relationships, not a grid with a fixed
number of shapes per row, so direction hints (`Rel_U`, `Rel_L`) and `UpdateLayoutConfig` do not affect placement; the
title is a frame around the diagram; a person is a field with semicircular ends, without an icon; an external element is
distinguished by a dashed line and a lighter background, not by colour; the element kind is written as in Mermaid
(`[Person]`, `[Software System]`); all five headers are drawn the same way.

## Git graph (`gitGraph`)

| Element | Syntax | Requirement |
|---|---|---|
| commit | `commit`, `commit id: "a" tag: "v1" type: HIGHLIGHT` | GIT-001 |
| branch | `branch nazwa`, `checkout nazwa`, `switch nazwa` | GIT-002 |
| merge, cherry-pick | `merge nazwa tag: "v2"`; `cherry-pick id: "a"` | GIT-002 |
| accessibility | `accTitle`, `accDescr` | GIT-001 |

Vertical orientation (`gitGraph TB:`, `BT:`, GIT-005): branches are columns, and a commit's label stands on its right
side. Deliberate differences from Mermaid: a commit without a given identifier has no label (Mermaid generates a random
identifier, so the drawing would not be reproducible); the identifier stands horizontally below the commit, and the gap
between two commits fits their own labels; branches stand in order of creation (`order:` is ignored); by default the
drawing is neutral, `WithColors` gives the branches palette colours.

## Block diagram (`block-beta`, `block`)

| Element | Syntax | Requirement |
|---|---|---|
| grid | `columns 3`, `columns auto` | BLK-001 |
| block | `id`, `id["Etykieta"]`, with a width `id:2`; `space`, `space:2` | BLK-001 |
| shapes | `(…)`, `([…])`, `((…))`, `{…}`, `{{…}}`, `[(…)]`, arrow `id<["…"]>(right)` | BLK-002 |
| composite block | `block:id:3 … end` | BLK-002 |
| connections | `a --> b`, `a --- b`, `a -.-> b`, `a -- "tekst" --> b` | BLK-003 |
| styling | `style`, `classDef`, `class` | BLK-003 |

Deliberate differences from Mermaid: a connection is a straight segment between the edges of the shapes and may pass over
another block; when the diagram has connections, the gaps between blocks grow so that arrowheads and labels fit;
a block arrow with the `y` direction is drawn pointing right.

## Architecture diagram (`architecture-beta`, `architecture`)

| Element | Syntax | Requirement |
|---|---|---|
| service | `service id(ikona)[Podpis]`, with `in grupa` | ARC-001 |
| group | `group id(ikona)[Podpis]`, with `in grupa` | ARC-001 |
| junction | `junction id` | ARC-001 |
| edge | `a:R -- L:b`, `a:B --> T:b`, `a:R <-[tekst]-> L:b`, `a{group}:R -- L:b` | ARC-002 |
| title, accessibility | `title Tekst`; `accTitle`, `accDescr` | ARC-001 |

Deliberate differences from Mermaid: the icons are this library's own simplified symbols for the five built-in names
(`database`, `server`, `disk`, `cloud`, `internet`); an icon with any other name, including from external icon sets, is
an empty field; a group's icon is not drawn, the group has only its label; the position of services follows solely from
the edge sides, and a service outside a group that would land within the group's frame moves further in its direction;
an edge is a polyline with at most two bends and may pass over another service.
