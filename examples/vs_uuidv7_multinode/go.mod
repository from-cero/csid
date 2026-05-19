module github.com/from-cero/cero-id/examples/vs_uuidv7_multinode

go 1.25.5

require (
	github.com/HdrHistogram/hdrhistogram-go v1.2.0
	github.com/from-cero/cero-id v0.0.0
	github.com/google/uuid v1.6.0
)

replace github.com/from-cero/cero-id => ../..
