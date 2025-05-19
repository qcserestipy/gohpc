package main

import (
	"flag"
	"math/big"
	"runtime"
	"time"

	"github.com/qcserestipy/gohpc/pkg/workerpool"
	"github.com/sirupsen/logrus"
)

func init() {
	formatter := &logrus.TextFormatter{}
	formatter.FullTimestamp = true
	formatter.TimestampFormat = time.RFC3339
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(formatter)
}

// Task defines the range of k-values for the Chudnovsky series.
type Task struct {
	Start int
	End   int
}

func main() {
	termsPtr := flag.Int("terms", 10, "Number of terms in the Chudnovsky series")
	digitsPtr := flag.Int("digits", 50, "Number of decimal places to print for π")
	flag.Parse()
	N := *termsPtr
	digits := *digitsPtr

	logrus.Infof("Computing π with %d terms and printing %d decimal places", N, digits)

	guardBits := uint(64)
	prec := uint(float64(digits)*3.32) + guardBits

	// Precompute factorial table up to 6*N
	maxFact := 6 * N
	logrus.Infof("Precomputing factorials up to %d...", maxFact)
	facts := make([]*big.Int, maxFact+1)
	facts[0] = big.NewInt(1)
	for i := 1; i <= maxFact; i++ {
		facts[i] = new(big.Int).Mul(facts[i-1], big.NewInt(int64(i)))
	}

	numWorkers := runtime.NumCPU()
	logrus.Infof("Using %d CPU cores", numWorkers)
	numTasks := numWorkers * 8
	tasks := make([]Task, numTasks)
	chunk := N / numTasks
	remainder := N % numTasks
	for i := 0; i < numTasks; i++ {
		start := i*chunk + min(i, remainder)
		count := chunk
		if i < remainder {
			count++
		}
		tasks[i] = Task{Start: start, End: start + count}
	}

	pool := workerpool.New[Task, *big.Float](numWorkers)
	work := func(t Task) *big.Float {
		sum := new(big.Float).SetPrec(prec)
		for k := t.Start; k < t.End; k++ {
			// Numerator: (6k)! * (13591409 + 545140134*k)
			num := new(big.Int).Mul(facts[6*k], big.NewInt(int64(13591409+545140134*k)))

			// Denominator: (3k)! * (k!)^3 * (-640320)^(3k)
			d1 := facts[3*k]
			d2 := new(big.Int).Exp(facts[k], big.NewInt(3), nil)
			d3 := new(big.Int).Exp(big.NewInt(-640320), big.NewInt(int64(3*k)), nil)
			den := new(big.Int).Mul(d1, d2)
			den.Mul(den, d3)

			// Convert to big.Float and add to sum
			nf := new(big.Float).SetPrec(prec).SetInt(num)
			df := new(big.Float).SetPrec(prec).SetInt(den)
			term := new(big.Float).Quo(nf, df)
			sum.Add(sum, term)
		}
		return sum
	}

	startTime := time.Now()
	partials := pool.Run(tasks, work)
	totalSum := new(big.Float).SetPrec(prec)
	for _, part := range partials {
		totalSum.Add(totalSum, part)
	}

	// C = 426880 * sqrt(10005)
	sqrtArg := new(big.Float).SetPrec(prec).SetInt(big.NewInt(10005))
	root := new(big.Float).SetPrec(prec).Sqrt(sqrtArg)
	factor := new(big.Float).SetPrec(prec).SetInt(big.NewInt(426880))
	C := new(big.Float).SetPrec(prec).Mul(factor, root)

	// π = C / totalSum
	pi := new(big.Float).SetPrec(prec).Quo(C, totalSum)
	elapsed := time.Since(startTime)

	// Reference π string with 500 digits
	const realPiStr = "3.141592653589793238462643383279502884197169399375105820974944592307816406286208998628034825342117067982148086513282306647093844609550582231725359408128481117450284102701938521105559644622948954930381964428810975665933446128475648233786783165271201909145648566923460348610454326648213393607260249141273724587006606315588174881520920962829254091715364367892590360011330530548820466521384146951941511609433057270365759591953092186117381932611793105118548074462379962749567351885752724891227938183011949129833673362440656643086021394946395224737190702179860943702770539217176293176752384674818467669405132000568127145263560827785771342757789609173637178721468440901224953430146549585371050792279689258923542019956112129021960864034418159813629774771309960518707211349999998372978049951059731732816096318595024459455346908302642522308253344685035261931188171010003137838752886587533208381420617177669147303598253490428755468731159562863882353787593751957781857780532171226806613001927876611195909216420198"
	refPi := new(big.Float).SetPrec(prec)
	if _, ok := refPi.SetString(realPiStr); !ok {
		logrus.Fatal("failed to parse reference π")
	}

	// Compute absolute error: |pi - refPi|
	err := new(big.Float).SetPrec(prec).Sub(pi, refPi)
	err.Abs(err)

	// Print result
	piStr := pi.Text('f', digits)
	errStr := err.Text('f', digits)
	logrus.Infof("π ≈ %s", piStr)
	logrus.Infof("Absolute error ≈ %s", errStr)
	logrus.Infof("Computed in %s", elapsed)
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
