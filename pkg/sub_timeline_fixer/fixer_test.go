package sub_timeline_fixer

import (
	"testing"

	"github.com/james-bowman/nlp"
	"github.com/james-bowman/nlp/measures/pairwise"
	"gonum.org/v1/gonum/mat"
)

func TestTFIDF(t *testing.T) {
	testCorpus := []string{
		"The quick brown fox jumped over the lazy dog",
		"hey diddle diddle, the cat and the fiddle",
		"the cow jumped over the moon",
		"the little dog laughed to see such fun",
		"and the dish ran away with the spoon",
	}

	query := "the brown fox ran around the dog"

	vectoriser := nlp.NewCountVectoriser(EnStopWords...)
	transformer := nlp.NewTfidfTransformer()

	// set k (the number of dimensions following truncation) to 4
	reducer := nlp.NewTruncatedSVD(4)

	lsiPipeline := nlp.NewPipeline(vectoriser, transformer, reducer)

	// Transform the corpus into an LSI fitting the model to the documents in the process
	lsi, err := lsiPipeline.FitTransform(testCorpus...)
	if err != nil {
		t.Errorf("Failed to process documents because %v", err)
		return
	}

	// run the query through the same pipeline that was fitted to the corpus and
	// to project it into the same dimensional space
	queryVector, err := lsiPipeline.Transform(query)
	if err != nil {
		t.Errorf("Failed to process documents because %v", err)
		return
	}

	// iterate over document feature vectors (columns) in the LSI matrix and compare
	// with the query vector for similarity.  Similarity is determined by the difference
	// between the angles of the vectors known as the cosine similarity
	highestSimilarity := -1.0
	var matched int
	_, docs := lsi.Dims()
	for i := 0; i < docs; i++ {
		similarity := pairwise.CosineSimilarity(queryVector.(mat.ColViewer).ColView(0), lsi.(mat.ColViewer).ColView(i))
		if similarity > highestSimilarity {
			matched = i
			highestSimilarity = similarity
		}
	}

	t.Logf("\n\nMatched '%s'", testCorpus[matched])
	// Output: Matched 'The quick brown fox jumped over the lazy dog'
}

//var timelineFixer = NewSubTimelineFixer(sub_timeline_fiexer.SubTimelineFixerConfig{
//	// V1
//	V1_MaxCompareDialogue: 3,
//	V1_MaxStartTimeDiffSD: 0.1,
//	V1_MinMatchedPercent:  0.1,
//	V1_MinOffset:          0.1,
//	// V2
//	V2_SubOneUnitProcessTimeOut: 5 * 60,
//	V2_FrontAndEndPerBase:       0.1,
//	V2_FrontAndEndPerSrc:        0.2,
//	V2_WindowMatchPer:           0.2,
//	V2_CompareParts:             3,
//	V2_FixThreads:               2,
//	V2_MaxStartTimeDiffSD:       0.1,
//	V2_MinOffset:                0.2,
//})
