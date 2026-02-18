package parser

import "testing"

func TestIsLikelyHeading_Multilingual(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
		lang     string
		reason   string
	}{
		// --- English ---
		{"Section 1 - Introduction", true, "en", "English section prefix"},
		{"Chapter 3 - Methods", true, "en", "English chapter prefix"},
		{"Article 5 - Obligations", true, "en", "English article prefix"},
		{"Part II - Analysis", true, "en", "English part prefix"},
		{"Figure 1 Summary of results", true, "en", "English figure + digit"},
		{"Figure out the problem", false, "en", "English 'figure' without digit"},

		// --- Spanish ---
		{"Sección 2 - Alcance", true, "es", "Spanish sección with accent"},
		{"Seccion 2 - Alcance", true, "es", "Spanish seccion without accent"},
		{"Capítulo 4 - Resultados", true, "es", "Spanish capítulo with accent"},
		{"Capitulo 4 - Resultados", true, "es", "Spanish capitulo without accent"},
		{"Anexo A - Diagramas", true, "es", "Spanish anexo"},
		{"Tabla 3 Especificaciones", true, "es", "Spanish tabla + digit"},
		{"Tabla de contenidos", false, "es", "Spanish 'tabla' without digit — body text"},
		{"Figura 5 Diagrama de flujo", true, "es", "Spanish figura + digit"},
		{"Cuadro 2 Resumen", true, "es", "Spanish cuadro + digit"},
		{"Gráfico 1 Tendencias", true, "es", "Spanish gráfico + digit"},

		// --- Portuguese ---
		{"Seção 1 - Introdução", true, "pt", "Portuguese seção with accent"},
		{"Secao 1 - Introdução", true, "pt", "Portuguese secao without accent"},
		{"Capítulo 2 - Metodologia", true, "pt", "Portuguese capítulo (same as Spanish)"},
		{"Artigo 3 - Disposições", true, "pt", "Portuguese artigo"},
		{"Anexo B - Tabelas", true, "pt", "Portuguese anexo (same as Spanish)"},
		{"Tabela 4 Dados experimentais", true, "pt", "Portuguese tabela + digit"},
		{"Tabela seguinte mostra", false, "pt", "Portuguese 'tabela' without digit — body text"},
		{"Figura 2 Gráfico de barras", true, "pt", "Portuguese figura + digit"},
		{"Quadro 1 Comparativo", true, "pt", "Portuguese quadro + digit"},

		// --- French ---
		{"Chapitre 1 - Introduction", true, "fr", "French chapitre"},
		{"Partie 2 - Analyse", true, "fr", "French partie"},
		{"Annexe C - Références", true, "fr", "French annexe"},
		{"Article 7 - Conditions", true, "fr", "French article"},
		{"Tableau 3 Récapitulatif", true, "fr", "French tableau + digit"},
		{"Tableau récapitulatif des", false, "fr", "French 'tableau' without digit — body text"},
		{"Figure 1 Schéma principal", true, "fr", "French figure + digit"},
		{"Graphique 2 Évolution", true, "fr", "French graphique + digit"},

		// --- Numbered sections (language-agnostic) ---
		{"1. Introduction", true, "any", "Numbered section 1."},
		{"3.9.1 Modelo A: AV Cabezal Standard", true, "any", "Deep numbered section"},
		{"7.3.1.2 Subsection detail", true, "any", "4-level numbered section"},
		{"10.2 Configuration", true, "any", "Double-digit numbered section"},

		// --- All-caps (language-agnostic) ---
		{"INTRODUCTION", true, "any", "All-caps heading"},
		{"CAPÍTULO 3 - ESPECIFICACIONES", true, "any", "All-caps Spanish heading"},
		{"CHAPITRE 2 - MÉTHODES", true, "any", "All-caps French heading"},
		{"ANEXOS", true, "any", "All-caps Spanish annex"},

		// --- Should NOT be detected as heading ---
		{"This is a normal paragraph of text that happens to be somewhat long but not a heading at all.", false, "en", "Regular paragraph"},
		{"En esta sección explicamos los resultados obtenidos durante las pruebas realizadas.", false, "es", "Spanish body text starting lowercase"},
		{"Nesta seção apresentamos os dados coletados ao longo do experimento.", false, "pt", "Portuguese body text starting lowercase"},
		{"Dans cette partie nous analysons les différentes approches possibles pour résoudre ce problème.", false, "fr", "French body text starting lowercase"},
		{"AB", false, "any", "Too short for all-caps heading (len <= 2)"},
		{"ok", false, "any", "Short lowercase"},
	}

	for _, tt := range tests {
		got := isLikelyHeading(tt.line)
		if got != tt.expected {
			t.Errorf("[%s] isLikelyHeading(%q) = %v, want %v (%s)",
				tt.lang, tt.line, got, tt.expected, tt.reason)
		}
	}
}

func TestSplitPageIntoSections_MultilingualHeadings(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		lang     string
		wantSecs int
		wantH    string // substring that should appear in some heading
	}{
		{
			name:     "Spanish numbered sections",
			text:     "3.1 Condiciones Ambientales\nTemperatura: 0-40°C\nHumedad: 10-90%\n3.2 Condiciones Eléctricas\nVoltaje: 220V",
			lang:     "es",
			wantSecs: 2,
			wantH:    "3.1 Condiciones",
		},
		{
			name:     "Portuguese sections",
			text:     "Seção 1 - Introdução\nEste documento descreve o sistema.\nArtigo 2 - Escopo\nO escopo inclui todos os componentes.",
			lang:     "pt",
			wantSecs: 2,
			wantH:    "Artigo 2",
		},
		{
			name:     "French sections",
			text:     "Chapitre 1 - Introduction\nCe document décrit le système.\nPartie 2 - Analyse\nL'analyse comprend les tests.",
			lang:     "fr",
			wantSecs: 2,
			wantH:    "Chapitre 1",
		},
		{
			name:     "Mixed language headings",
			text:     "SUMMARY\nThis is the summary.\nAnexo A - Diagramas\nDiagram details here.\nTableau 1 Résultats\nData row 1",
			lang:     "mixed",
			wantSecs: 3,
			wantH:    "Anexo A",
		},
		{
			name:     "French table/figure with digit guard",
			text:     "Tableau récapitulatif des résultats montrant l'évolution.\nTableau 1 Résultats principaux\nDonnées ici.",
			lang:     "fr",
			wantSecs: 2, // "Tableau récapitulatif..." is body, "Tableau 1" is heading
			wantH:    "Tableau 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := splitPageIntoSections(tt.text, 1)

			if len(sections) != tt.wantSecs {
				t.Errorf("[%s] got %d sections, want %d", tt.lang, len(sections), tt.wantSecs)
				for i, s := range sections {
					t.Logf("  [%d] heading=%q content=%.80s", i, s.Heading, s.Content)
				}
			}

			found := false
			for _, s := range sections {
				if contains(s.Heading, tt.wantH) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("[%s] no section heading contains %q", tt.lang, tt.wantH)
				for i, s := range sections {
					t.Logf("  [%d] heading=%q", i, s.Heading)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
