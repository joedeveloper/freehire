package main

import "strings"

// emitVocab renders one closed vocabulary as a frozen value array plus a string-union
// type, e.g. emitVocab("Seniority", "SENIORITY_VALUES", […]) →
//
//	export const SENIORITY_VALUES = ['intern', …] as const;
//	export type Seniority = (typeof SENIORITY_VALUES)[number];
func emitVocab(typeName, constName string, values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "'" + v + "'"
	}
	var b strings.Builder
	b.WriteString("export const " + constName + " = [" + strings.Join(quoted, ", ") + "] as const;\n")
	b.WriteString("export type " + typeName + " = (typeof " + constName + ")[number];\n")
	return b.String()
}
