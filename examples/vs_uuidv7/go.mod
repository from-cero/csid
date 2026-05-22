module github.com/from-cero/csid/examples/vs_uuidv7

go 1.25.5

require (
	github.com/HdrHistogram/hdrhistogram-go v1.2.0
	github.com/from-cero/csid v0.0.0
	github.com/google/uuid v1.6.0
)

replace github.com/from-cero/csid => ../..
