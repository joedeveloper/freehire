package telegram

import "testing"

func TestLooksLikeVacancy(t *testing.T) {
	pass := []struct{ name, text string }{
		{"ru marker вакансия", "Вакансия: Go разработчик в финтех"},
		{"ru marker ищем", "Ищем SEO-специалиста в криптопроект (удаленно)"},
		{"ru salary", "Платят 250 000 руб на руки, офис в Москве"},
		{"en hiring", "We are hiring a senior backend engineer"},
		{"en salary range", "ML & full-stack engineers, $110k-220k + bonus + equity, London"},
		{"board template", "Senior Fullstack Engineer\n#удаленка #senior\nCompany: RugsDotFun\nSalary: $120k - $200k"},
		{"required experience", "Требуется опыт от 2 лет, стек: Go, Postgres. Резюме в личку"},
		{"recall bias: weak but plausible", "Команде нужен продакт. Подробности у @someone"},
	}
	for _, tc := range pass {
		t.Run("pass/"+tc.name, func(t *testing.T) {
			if !LooksLikeVacancy(tc.text) {
				t.Errorf("filtered out a plausible vacancy: %q", tc.text)
			}
		})
	}

	reject := []struct{ name, text string }{
		{"meme", "Пятница! Всем хороших выходных 🎉"},
		{"news digest", "Дайджест новостей недели: Яндекс выпустил новую модель, OpenAI снова в суде"},
		{"course ad", "Скидка 50% на курс по Python до конца недели! Успей записаться"},
		{"empty-ish", "🔥🔥🔥"},
	}
	for _, tc := range reject {
		t.Run("reject/"+tc.name, func(t *testing.T) {
			if LooksLikeVacancy(tc.text) {
				t.Errorf("let through an obvious non-vacancy: %q", tc.text)
			}
		})
	}
}
