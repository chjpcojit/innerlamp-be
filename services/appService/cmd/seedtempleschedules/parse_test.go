package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

const goodFile = `{
  "meta": {"format": 1, "source": "mock"},
  "items": [
    {"id": "s1", "temple_id": "t1", "name": "Lễ Dược Sư", "start_time": "08:00", "end_time": "11:00",
     "weekdays": [0], "description": "d", "instructor": "i", "location": "l", "preparation": "p", "sort_order": 2,
     "steps": [{"position": 1, "time": "08:00", "text": "Niệm Phật"}, {"position": 2, "time": null, "text": "Hồi hướng"}]},
    {"id": "s2", "temple_id": "t1", "name": "Công phu", "start_time": "04:00"}
  ]
}`

func TestParseCatalogue_Valid(t *testing.T) {
	c, rep, err := parseCatalogue([]byte(goodFile))
	require.NoError(t, err)
	assert.Equal(t, "mock", c.Source)
	assert.Equal(t, 0, rep.Normalized)
	require.Len(t, c.Items, 2)

	full := c.Items[0]
	assert.Equal(t, domain.TimeOfDay(480), full.StartTime)
	require.NotNil(t, full.EndTime)
	assert.Equal(t, domain.TimeOfDay(660), *full.EndTime)
	assert.Equal(t, []int{0}, full.Weekdays)
	require.Len(t, full.Steps, 2)
	require.NotNil(t, full.Steps[0].Time)
	assert.Nil(t, full.Steps[1].Time)

	min := c.Items[1]
	assert.Nil(t, min.EndTime)
	assert.Empty(t, min.Steps)
	assert.NoError(t, c.Verify())
}

func TestParseCatalogue_NormalisesNFC(t *testing.T) {
	// "ệ" as e + combining dot below + combining circumflex (NFD).
	src := `{"meta":{"format":1,"source":"mock"},"items":[{"id":"s","temple_id":"t","name":"Lệ","start_time":"04:00"}]}`
	c, rep, err := parseCatalogue([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Normalized)
	assert.Equal(t, "Lệ", c.Items[0].Name)
}

func TestParseCatalogue_Rejects(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"broken json", `{`, "decode"},
		{"unknown item key", `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"04:00","weekday":[1]}]}`, "unknown field"},
		{"unknown meta key", `{"meta":{"format":1,"source":"m","x":1},"items":[]}`, "unknown field"},
		{"unknown step key", `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"04:00","steps":[{"position":1,"text":"a","note":"b"}]}]}`, "unknown field"},
		{"trailing data", goodFile + `{}`, "unexpected data"},
		{"wrong format", `{"meta":{"format":2,"source":"m"},"items":[]}`, "meta.format"},
		{"missing source", `{"meta":{"format":1},"items":[]}`, "meta.source"},
		{"bad start time", `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"7:5"}]}`, "start_time"},
		{"bad end time", `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"04:00","end_time":"24:00"}]}`, "end_time"},
		{"bad step time", `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"04:00","steps":[{"position":1,"time":"99:00","text":"a"}]}]}`, "step 1 time"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseCatalogue([]byte(tc.src))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseCatalogue_VerifyCatchesMissingPositionAndEmptyText(t *testing.T) {
	src := `{"meta":{"format":1,"source":"m"},"items":[{"id":"s","temple_id":"t","name":"n","start_time":"04:00","steps":[{"text":"a"},{"position":1,"text":""}]}]}`
	c, _, err := parseCatalogue([]byte(src))
	require.NoError(t, err)
	err = c.Verify()
	require.ErrorIs(t, err, domain.ErrInvalidTempleSchedule)
	assert.Contains(t, err.Error(), "position must be >= 1")
	assert.Contains(t, err.Error(), "text is empty")
}
