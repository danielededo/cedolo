package extract

// Distances the extractor reads a page with. Payslips are laid out on A4 in
// points, with text from about 5 to 12 points tall: fixed distances are in
// points, the others in multiples of the text height, so that they follow
// the type size.

// In points.
const (
	// lineTolerance is the largest difference between the tops of two
	// tokens on the same printed line.
	lineTolerance = 2.5
	// wideGap ends a label's value beside it, and ends the item table when
	// it separates two rows: a gap this wide is another column or block.
	wideGap = 40.0
	// stackedGap is the most a label's second line, or the qualifier line
	// of a leave table, sits below the first.
	stackedGap = 6.0
	// labelSlack is how far a label's second line, or a qualifier printed
	// over it, may stand to its left or right and still belong to it.
	labelSlack = 5.0
	// cellMargin widens a cell to the left of its label, for values
	// printed a little before it.
	cellMargin = 2.0
	// numericMargin widens the numeric columns of the item table to the
	// left, for right-aligned amounts longer than their header.
	numericMargin = 25.0
)

// In multiples of the text height.
const (
	// belowReach is how far below a label its value may be printed.
	belowReach = 3.0
	// farReach is how far below a label a value may still be printed in a
	// tall box, as a total over a list of contributions. Such a value is
	// only an alternative reading, taken when the payslip adds up with it.
	farReach = 6.0
	// qualifierReach is how far above a label a word qualifying it may be
	// printed, as "GG." over "CCNL".
	qualifierReach = 1.6
	// baselineSlack is how far two tokens' baselines may differ and still
	// be on the same printed row.
	baselineSlack = 0.2
	// wordGap is the widest gap between two words of one label.
	wordGap = 1.2
)
