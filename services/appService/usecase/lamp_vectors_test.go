package usecase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// lampRulesDir holds the cross-platform lamp vectors of the product spec
// (docs/product/lamp/rules/). They are read as-is, never copied, so a changed
// vector fails here first.
// VI: lampRulesDir chứa bộ vector dùng chung mọi nền tảng của product spec Thắp
// đèn. Test đọc thẳng file, không chép lại, nên vector đổi là test ở đây báo trước.
var lampRulesDir = filepath.Join("..", "..", "..", "docs", "product", "lamp", "rules")

// lampVectorCase is one entry of a rules file's "cases" array.
// VI: lampVectorCase là một phần tử của mảng "cases" trong file rules.
type lampVectorCase[In, Out any] struct {
	Name     string `json:"name"`
	Input    In     `json:"input"`
	Expected Out    `json:"expected"`
}

func loadLampVectors[In, Out any](t *testing.T, file string) []lampVectorCase[In, Out] {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(lampRulesDir, file))
	require.NoError(t, err)
	var doc struct {
		Cases []lampVectorCase[In, Out] `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc), "decode %s", file)
	require.NotEmpty(t, doc.Cases, "%s has no cases", file)
	return doc.Cases
}

// vectorText is a text value of the vector files: JSON null, a plain string,
// {"repeat": s, "times": n} (s repeated n times) or {"concat": [...]} (its
// parts joined).
// VI: vectorText là một giá trị chuỗi trong file vector: null, chuỗi thường,
// {"repeat": s, "times": n} (lặp s n lần) hoặc {"concat": [...]} (nối các phần).
type vectorText struct {
	present bool
	value   string
}

func (v *vectorText) UnmarshalJSON(b []byte) error {
	s, present, err := decodeVectorText(b)
	if err != nil {
		return err
	}
	v.present, v.value = present, s
	return nil
}

func decodeVectorText(b []byte) (string, bool, error) {
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return "", false, err
	}
	return expandVectorText(raw)
}

func expandVectorText(raw any) (string, bool, error) {
	switch v := raw.(type) {
	case nil:
		return "", false, nil
	case string:
		return v, true, nil
	case map[string]any:
		if parts, ok := v["concat"].([]any); ok {
			var sb strings.Builder
			for _, p := range parts {
				s, _, err := expandVectorText(p)
				if err != nil {
					return "", false, err
				}
				sb.WriteString(s)
			}
			return sb.String(), true, nil
		}
		s, okS := v["repeat"].(string)
		n, okN := v["times"].(float64)
		if okS && okN {
			return strings.Repeat(s, int(n)), true, nil
		}
	}
	b, _ := json.Marshal(raw)
	return "", false, &json.UnsupportedValueError{Str: string(b)}
}

// ptr is the request form of v: nil when the vector has null.
// VI: ptr là dạng request của v: nil khi vector ghi null.
func (v vectorText) ptr() *string {
	if !v.present {
		return nil
	}
	s := v.value
	return &s
}

// lightForVector runs Light with a mock repository and returns the lamp the use
// case handed over, or the validation error.
// VI: lightForVector chạy Light với repository giả và trả về đèn usecase chuyển
// xuống, hoặc lỗi validate.
func lightForVector(t *testing.T, now time.Time, in domain.LampInput) (*domain.Lamp, error) {
	t.Helper()
	repo := new(domain.MockLampRepository)
	var got domain.LightLampParams
	repo.On("Light", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(1).(domain.LightLampParams) }).
		Return(&domain.Lamp{ID: 1}, true, nil).Maybe()
	u := NewLampUsecase(repo, testLampConfig())
	u.now = func() time.Time { return now }

	if _, _, err := u.Light(context.Background(), 7, in); err != nil {
		return nil, err
	}
	return &got.Lamp, nil
}

// TestLampVectors_InputValidation runs lamp-input-validation.json (BR-LMP-01,
// 02, 06, 07). Lengths are counted after NFC, so expected texts are compared in
// NFC too.
// VI: Chạy lamp-input-validation.json. Độ dài đếm sau NFC nên chuỗi kỳ vọng cũng
// được so sánh ở dạng NFC.
func TestLampVectors_InputValidation(t *testing.T) {
	type input struct {
		LampType     string     `json:"lamp_type"`
		Intention    string     `json:"intention"`
		Recipient    vectorText `json:"recipient"`
		Prayer       vectorText `json:"prayer"`
		DurationDays int        `json:"duration_days"`
	}
	type expected struct {
		Valid       bool       `json:"valid"`
		Field       string     `json:"field"`
		Recipient   vectorText `json:"recipient"`
		Prayer      vectorText `json:"prayer"`
		DedicatedTo string     `json:"dedicated_to"`
	}
	for _, c := range loadLampVectors[input, expected](t, "lamp-input-validation.json") {
		t.Run(c.Name, func(t *testing.T) {
			lamp, err := lightForVector(t, lampNow, domain.LampInput{
				LampType: c.Input.LampType, Intention: c.Input.Intention,
				Recipient: c.Input.Recipient.ptr(), Prayer: c.Input.Prayer.ptr(),
				DurationDays: c.Input.DurationDays,
			})

			if !c.Expected.Valid {
				assertInvalidLampField(t, err, c.Expected.Field)
				return
			}
			require.NoError(t, err)
			assertVectorText(t, "recipient", c.Expected.Recipient, lamp.Recipient)
			assertVectorText(t, "prayer", c.Expected.Prayer, lamp.Prayer)
			if c.Expected.DedicatedTo == "all_beings" {
				assert.Nil(t, lamp.Recipient, "a lamp for all beings is stored without a recipient")
			}
		})
	}
}

func assertVectorText(t *testing.T, field string, want vectorText, got *string) {
	t.Helper()
	if !want.present {
		assert.Nil(t, got, "%s must be null", field)
		return
	}
	require.NotNil(t, got, "%s must not be null", field)
	assert.Equal(t, norm.NFC.String(want.value), *got, field)
}

// TestLampVectors_Lifetime runs lamp-lifetime.json (BR-LMP-03, 04): end_at is
// lit_at + days×24h, and a lamp is lit while now is earlier than end_at — the
// same strict comparison as the repository's end_at > now.
// VI: Chạy lamp-lifetime.json: end_at = lit_at + số ngày×24h, đèn còn sáng khi now
// còn trước end_at — cùng phép so sánh chặt end_at > now của repository.
func TestLampVectors_Lifetime(t *testing.T) {
	type input struct {
		LitAt        time.Time `json:"lit_at"`
		DurationDays int       `json:"duration_days"`
		Now          time.Time `json:"now"`
	}
	type expected struct {
		EndAt time.Time `json:"end_at"`
		IsLit bool      `json:"is_lit"`
	}
	for _, c := range loadLampVectors[input, expected](t, "lamp-lifetime.json") {
		t.Run(c.Name, func(t *testing.T) {
			lamp, err := lightForVector(t, c.Input.LitAt, domain.LampInput{
				LampType: "OIL_LAMP", Intention: "PEACE", DurationDays: c.Input.DurationDays,
			})

			require.NoError(t, err)
			assert.True(t, c.Expected.EndAt.Equal(lamp.EndAt), "end_at: want %s, got %s", c.Expected.EndAt, lamp.EndAt)
			assert.Equal(t, c.Expected.IsLit, lamp.EndAt.After(c.Input.Now), "is_lit")
		})
	}
}

// TestLampVectors_LighterName runs lamp-lighter-name.json (BR-LMP-09): the name
// copied at lighting time and how the garden shows it.
// VI: Chạy lamp-lighter-name.json: tên chép lúc thắp và cách vườn đèn hiển thị.
func TestLampVectors_LighterName(t *testing.T) {
	type input struct {
		DharmaName *string `json:"dharma_name"`
		HideName   bool    `json:"hide_name"`
	}
	type expected struct {
		DisplayedName string `json:"displayed_name"`
	}
	for _, c := range loadLampVectors[input, expected](t, "lamp-lighter-name.json") {
		t.Run(c.Name, func(t *testing.T) {
			stored := domain.LighterNameAtLighting(c.Input.DharmaName, c.Input.HideName)
			if c.Input.HideName {
				assert.Nil(t, stored, "an anonymous lamp keeps no name")
			}

			assert.Equal(t, c.Expected.DisplayedName, domain.GardenLamp{LighterName: stored}.DisplayName())
			assert.Equal(t, c.Expected.DisplayedName, domain.Lamp{LighterName: stored}.DisplayName())
		})
	}
}
