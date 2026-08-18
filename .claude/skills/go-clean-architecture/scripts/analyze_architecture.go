package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// AnalysisResult holds the findings from architecture analysis
type AnalysisResult struct {
	File       string
	Line       int
	LayerFrom  string
	LayerTo    string
	Severity   string
	Finding    string
	RuleName   string
}

// ArchitectureAnalyzer analyzes Go projects for clean architecture violations
type ArchitectureAnalyzer struct {
	projectPath string
	results     []AnalysisResult
}

// NewArchitectureAnalyzer creates a new analyzer for the given project path
func NewArchitectureAnalyzer(projectPath string) *ArchitectureAnalyzer {
	return &ArchitectureAnalyzer{
		projectPath: projectPath,
		results:     []AnalysisResult{},
	}
}

// AnalyzeProject analyzes the entire project for violations
func (a *ArchitectureAnalyzer) AnalyzeProject() error {
	return filepath.Walk(a.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return a.analyzeFile(path)
	})
}

// AnalyzeFile analyzes a single Go file for violations
func (a *ArchitectureAnalyzer) analyzeFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil // Skip files that can't be parsed
	}

	relPath := getRelativePath(a.projectPath, filePath)
	sourceLayer := extractLayer(relPath)

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		importedLayer := extractImportedLayer(a.projectPath, importPath)

		// Check for violations
		if violation := checkViolation(sourceLayer, importedLayer, importPath); violation != "" {
			a.results = append(a.results, AnalysisResult{
				File:      relPath,
				Line:      fset.Position(imp.Pos()).Line,
				LayerFrom: sourceLayer,
				LayerTo:   importedLayer,
				Severity:  getSeverity(sourceLayer, importedLayer),
				Finding:   violation,
				RuleName:  getRuleName(sourceLayer, importedLayer),
			})
		}
	}

	return nil
}

// extractLayer determines which layer a file belongs to
func extractLayer(filePath string) string {
	parts := strings.Split(filePath, string(os.PathSeparator))
	for _, part := range parts {
		switch part {
		case "domain":
			return "domain"
		case "repository":
			return "repository"
		case "usecase":
			return "usecase"
		case "delivery":
			return "delivery"
		case "infrastructure":
			return "infrastructure"
		case "adapter":
			return "adapter"
		}
	}
	return "unknown"
}

// extractImportedLayer determines which layer is being imported
func extractImportedLayer(projectPath, importPath string) string {
	// Handle internal imports
	parts := strings.Split(importPath, "/")
	for _, part := range parts {
		switch part {
		case "domain":
			return "domain"
		case "repository":
			return "repository"
		case "usecase":
			return "usecase"
		case "delivery":
			return "delivery"
		case "infrastructure":
			return "infrastructure"
		case "adapter":
			return "adapter"
		}
	}
	return "external"
}

// checkViolation checks if an import violates clean architecture rules
func checkViolation(sourceLayer, targetLayer, importPath string) string {
	// Rule: Domain should not import other internal layers
	if sourceLayer == "domain" {
		if targetLayer != "external" && targetLayer != "unknown" {
			return fmt.Sprintf("Domain imports %s layer", targetLayer)
		}
	}

	// Rule: Usecase should not import delivery or concrete implementations
	if sourceLayer == "usecase" {
		if targetLayer == "delivery" {
			return "Usecase imports delivery layer"
		}
		if targetLayer == "repository" && !strings.Contains(importPath, "domain") {
			return "Usecase imports concrete repository (should use domain interfaces)"
		}
		if targetLayer == "infrastructure" {
			return "Usecase imports infrastructure layer directly"
		}
	}

	// Rule: Repository should not import delivery or usecase
	if sourceLayer == "repository" {
		if targetLayer == "delivery" {
			return "Repository imports delivery layer"
		}
		if targetLayer == "usecase" {
			return "Repository imports usecase layer"
		}
	}

	// Rule: Delivery should not import repository concrete types
	if sourceLayer == "delivery" {
		if targetLayer == "repository" {
			return "Delivery imports concrete repository (should use domain interfaces)"
		}
	}

	return ""
}

// getSeverity returns the severity level of a violation
func getSeverity(sourceLayer, targetLayer string) string {
	// Critical violations
	if sourceLayer == "domain" {
		return "CRITICAL"
	}
	if sourceLayer == "repository" && targetLayer == "delivery" {
		return "CRITICAL"
	}
	if sourceLayer == "usecase" && targetLayer == "delivery" {
		return "HIGH"
	}

	return "HIGH"
}

// getRuleName returns the name of the violated rule
func getRuleName(sourceLayer, targetLayer string) string {
	return fmt.Sprintf("LayerBoundary_%s_to_%s", sourceLayer, targetLayer)
}

// getRelativePath returns the relative path from project root
func getRelativePath(projectPath, filePath string) string {
	rel, err := filepath.Rel(projectPath, filePath)
	if err != nil {
		return filePath
	}
	return rel
}

// PrintResults prints the analysis results
func (a *ArchitectureAnalyzer) PrintResults() {
	if len(a.results) == 0 {
		fmt.Println("✅ No clean architecture violations found!")
		return
	}

	fmt.Printf("❌ Found %d clean architecture violations:\n\n", len(a.results))

	// Group by severity
	critical := []AnalysisResult{}
	high := []AnalysisResult{}
	medium := []AnalysisResult{}

	for _, r := range a.results {
		switch r.Severity {
		case "CRITICAL":
			critical = append(critical, r)
		case "HIGH":
			high = append(high, r)
		default:
			medium = append(medium, r)
		}
	}

	if len(critical) > 0 {
		fmt.Println("🔴 CRITICAL VIOLATIONS:")
		for _, r := range critical {
			fmt.Printf("  %s:%d\n", r.File, r.Line)
			fmt.Printf("    %s → %s\n", r.LayerFrom, r.LayerTo)
			fmt.Printf("    Finding: %s\n", r.Finding)
			fmt.Printf("    Rule: %s\n\n", r.RuleName)
		}
	}

	if len(high) > 0 {
		fmt.Println("🟠 HIGH VIOLATIONS:")
		for _, r := range high {
			fmt.Printf("  %s:%d\n", r.File, r.Line)
			fmt.Printf("    %s → %s\n", r.LayerFrom, r.LayerTo)
			fmt.Printf("    Finding: %s\n", r.Finding)
			fmt.Printf("    Rule: %s\n\n", r.RuleName)
		}
	}

	if len(medium) > 0 {
		fmt.Println("🟡 MEDIUM VIOLATIONS:")
		for _, r := range medium {
			fmt.Printf("  %s:%d\n", r.File, r.Line)
			fmt.Printf("    %s → %s\n", r.LayerFrom, r.LayerTo)
			fmt.Printf("    Finding: %s\n", r.Finding)
			fmt.Printf("    Rule: %s\n\n", r.RuleName)
		}
	}
}

func main() {
	var targetPath string
	flag.StringVar(&targetPath, "path", ".", "Path to analyze (file or directory)")
	flag.Parse()

	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	analyzer := NewArchitectureAnalyzer(targetPath)

	if info.IsDir() {
		if err := analyzer.AnalyzeProject(); err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing project: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := analyzer.analyzeFile(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing file: %v\n", err)
			os.Exit(1)
		}
	}

	analyzer.PrintResults()

	if len(analyzer.results) > 0 {
		os.Exit(1)
	}
}
