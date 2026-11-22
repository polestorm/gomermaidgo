# Wymagania: wykres słupkowy i liniowy


| Id | Wymaganie |
|---|---|
| XY-001 | Źródło zaczynające się od `xychart-beta` (albo `xychart`) daje wykres; `title` nadaje tytuł, w cudzysłowach albo bez; komentarze `%%` i puste linie są pomijane; `accTitle` i `accDescr` trafiają do `title` i `desc`. |
| XY-002 | Oś pozioma: `x-axis [a, b, c]` z opcjonalnym tytułem przed listą albo `x-axis tytuł min --> max`. Oś pionowa: `y-axis tytuł min --> max`, sam tytuł albo sam zakres. Serie: `bar [..]` i `line [..]`, z opcjonalną nazwą przed listą; każda seria ma tyle wartości, ile jest kategorii. |
| XY-003 | Rysunek: oś pionowa z okrągłą podziałką i liniami siatki, kategorie pod osią poziomą, słupek od zera do wartości, linia przez środki kategorii z punktami. Bez podanego zakresu oś pionowa obejmuje dane, a przy słupkach także zero. Gdy etykiety kategorii się nie mieszczą, zostaje co któraś. |
| XY-004 | Kilka serii słupków stoi obok siebie w obrębie kategorii; kolejne linie różnią się kreskowaniem, a kolejne serie słupków jasnością, więc wykres czyta się bez koloru. Serie z nazwami mają legendę pod wykresem. Wartość ujemna jest rysowana pod osią zera. |
| XY-005 | Element spoza profilu albo błąd składni daje błąd z numerem linii i stałym kodem: zły nagłówek, nierozpoznana linia, wartość niebędąca liczbą, zakres malejący, seria o innej liczbie wartości, przekroczenie limitów (200 wartości, 12 serii), brak serii. |
| XY-006 | `xychart-beta horizontal` daje wykres poziomy: kategorie stoją z góry na dół przy osi pionowej, wartości biegną wzdłuż osi poziomej z podziałką pod wykresem, słupki rosną od zera w prawo, a linia łączy punkty kolejnych kategorii. Tytuł osi kategorii stoi z lewej strony, obrócony wzdłuż osi; tytuł osi wartości pod podziałką. |
