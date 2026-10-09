package styles

import (
	"testing"
)

func TestBaseStyle(t *testing.T) {
	if s := BaseStyle().Render("x"); s == "" {
		t.Fatal("BaseStyle should render non-empty")
	}
}

func TestRegular(t *testing.T) {
	if Regular().Render("x") == "" {
		t.Fatal("Regular should render non-empty")
	}
}

func TestBold(t *testing.T) {
	if !Bold().GetBold() {
		t.Fatal("Bold should have bold enabled")
	}
}

func TestPadded(t *testing.T) {
	if Padded().GetPaddingLeft() != 1 {
		t.Fatal("Padded should have horizontal padding")
	}
}

func TestBorders(t *testing.T) {
	for _, s := range []interface{}{Border(), ThickBorder(), DoubleBorder(), FocusedBorder(), DimBorder()} {
		_ = s
	}
	if Border().Render("x") == "" {
		t.Fatal("Border should render non-empty")
	}
	if ThickBorder().Render("x") == "" {
		t.Fatal("ThickBorder should render non-empty")
	}
	if DoubleBorder().Render("x") == "" {
		t.Fatal("DoubleBorder should render non-empty")
	}
	if FocusedBorder().Render("x") == "" {
		t.Fatal("FocusedBorder should render non-empty")
	}
	if DimBorder().Render("x") == "" {
		t.Fatal("DimBorder should render non-empty")
	}
}

func TestColorHelpers(t *testing.T) {
	for _, c := range []interface{}{PrimaryColor(), SecondaryColor(), AccentColor(), ErrorColor(), WarningColor(),
		SuccessColor(), InfoColor(), TextColor(), TextMutedColor(), TextEmphasizedColor(),
		BackgroundColor(), BackgroundSecondaryColor(), BackgroundDarkerColor(),
		BorderNormalColor(), BorderFocusedColor(), BorderDimColor()} {
		if c == nil {
			t.Fatal("color helper returned nil")
		}
	}
}

func TestImageBackground(t *testing.T) {
	if ImageBakcground == "" {
		t.Fatal("ImageBakcground should be set")
	}
}
