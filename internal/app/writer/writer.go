package writer

import (
	"context"
	"errors"
	"fmt"
	"github.com/kotche/bot/internal/model"
	"github.com/kotche/bot/internal/service/notes"
	"gopkg.in/telebot.v3"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	longProcessTimeout = 2
)

type Writer struct {
	bot   *telebot.Bot
	notes notes.Service
}

func New(bot *telebot.Bot, notes notes.Service) *Writer {
	return &Writer{bot: bot, notes: notes}
}

func (w *Writer) Start() {
	w.helpHandler()
	w.createNoteHandler()
	w.deleteHandler()
	w.getHandler()
	w.listNoteHandler()

	log.Println("Writer started...")
	w.bot.Start()
}

// helpHandler обработчик помощь
func (w *Writer) helpHandler() {
	helpMessage := "Доступные команды:\n" +
		"/new - создать новую заметку\n" +
		"/delete {id} - удалить заметку по id\n" +
		"/get {id} - получить заметку по id\n" +
		"/list - список заметок:\n" +
		"	| по-умолчанию выводит активные заметки\n" +
		"	| -a выводит все заметки (включая удаленные)\n" +
		"/help - показать это сообщение"

	w.bot.Handle("/help", func(c telebot.Context) error {
		return c.Send(helpMessage)
	})

	return
}

// createNoteHandler обработчик создать заметку
func (w *Writer) createNoteHandler() {
	var (
		currentNote      string
		selectedDateTime string
		selectedMonth    time.Month
		selectedDay      int
		isCreatingNote   bool
	)

	w.bot.Handle("/new", func(c telebot.Context) error {
		currentNote = ""
		selectedDateTime = ""
		selectedMonth = 0
		selectedDay = 0
		isCreatingNote = true
		return c.Send("Напечатайте текст заметки:", &telebot.ReplyMarkup{ForceReply: true})
	})

	w.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		if !isCreatingNote {
			return nil // Игнорируем любой текст, если не в процессе создания заметки
		}

		if currentNote == "" {
			currentNote = c.Text()
			markup := &telebot.ReplyMarkup{}
			markup.InlineKeyboard = [][]telebot.InlineButton{
				{
					telebot.InlineButton{Unique: "note_yes", Text: "Да"},
					telebot.InlineButton{Unique: "note_no", Text: "Нет"},
				},
			}
			return c.Send(fmt.Sprintf("Ваша заметка: \"%s\". Продолжить?", currentNote), markup)
		}
		if selectedMonth == 0 {
			month, err := strconv.Atoi(c.Text())
			if err != nil || month < 1 || month > 12 {
				return c.Send("Введите номер месяца (1-12):")
			}
			selectedMonth = time.Month(month)
			return w.sendDays(c, selectedMonth)
		}
		if selectedDay == 0 {
			day, err := strconv.Atoi(c.Text())
			if err != nil || day < 1 || day > 31 {
				return c.Send("Введите корректный день месяца:")
			}
			selectedDay = day
			return c.Send("Введите время в формате HH или HH:MM (например, 14 или 15:37):")
		}
		if selectedDateTime == "" {
			inputTime := c.Text()
			if !isValidTimeFormat(inputTime) {
				return c.Send("Введите корректное время в формате HH или HH:MM:")
			}
			year := time.Now().Year()
			selectedDateTime = fmt.Sprintf("%d-%02d-%02d %s", year, selectedMonth, selectedDay, formatTime(inputTime))
			markup := &telebot.ReplyMarkup{}
			markup.InlineKeyboard = [][]telebot.InlineButton{
				{
					telebot.InlineButton{Unique: "save_yes", Text: "Да"},
					telebot.InlineButton{Unique: "save_no", Text: "Нет"},
				},
			}
			return c.Send(fmt.Sprintf("Сохранить заметку: \"%s\" с напоминанием на %s?", currentNote, selectedDateTime), markup)
		}
		return nil
	})

	w.bot.Handle(&telebot.InlineButton{Unique: "note_yes"}, func(c telebot.Context) error {
		return c.Send("Когда напомнить? Введите номер месяца (1-12):")
	})

	w.bot.Handle(&telebot.InlineButton{Unique: "select_day"}, func(c telebot.Context) error {
		day, err := strconv.Atoi(c.Data())
		if err != nil || day < 1 || day > 31 {
			return c.Send("Ошибка при выборе дня. Попробуйте ещё раз.")
		}

		selectedDay = day
		return c.Send("Введите время в формате HH или HH:MM (например, 14 или 15:37):")
	})

	w.bot.Handle(&telebot.InlineButton{Unique: "note_no"}, func(c telebot.Context) error {
		currentNote = ""
		return c.Send("Напечатайте новую заметку:", &telebot.ReplyMarkup{ForceReply: true})
	})

	//Проверка юзера и сохранение заметки
	w.bot.Handle(&telebot.InlineButton{Unique: "save_yes"}, func(c telebot.Context) error {
		isCreatingNote = false

		ctx, cancel := context.WithTimeout(context.Background(), longProcessTimeout*time.Second)
		defer cancel()

		userID := model.UserID(c.Sender().ID)

		if err := w.notes.EnsureUserExists(ctx, model.User{
			ID:    userID,
			Login: c.Sender().Username},
		); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("context deadline exceeded while ensuring user '%d': %v", userID, err)
				return c.Send("Операция сохранения пользователя заняла слишком много времени. Попробуйте позже.")
			}
			log.Printf("failed to ensure user '%d' exists: %v", userID, err)
			return c.Send(fmt.Sprintf("Не удалось сохранить текущего пользователя '%d'", userID))
		}

		parsedTime, err := time.Parse("2006-01-02 15:04", selectedDateTime)
		if err != nil {
			log.Printf("failed to parse time '%s': %v", selectedDateTime, err)
			return c.Send("Ошибка при обработке даты и времени. Попробуйте ещё раз.")
		}

		noteID, err := w.notes.Create(ctx, model.Note{
			UserID:   userID,
			Text:     currentNote,
			NotifyAt: parsedTime,
		})
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("context deadline exceeded while creating note '%s' for user '%d': %v", currentNote, userID, err)
				return c.Send("Операция сохранения заметки заняла слишком много времени. Попробуйте позже.")
			}
			log.Printf("failed to create note '%s' for user '%d': %v", currentNote, userID, err)
			return c.Send(fmt.Sprintf("Не удалось сохранить заметку"))
		}

		return c.Send(fmt.Sprintf("Сохранена заметка \"%s\", id: %d. Напоминание %s.", currentNote, noteID, selectedDateTime))
	})

	w.bot.Handle(&telebot.InlineButton{Unique: "save_no"}, func(c telebot.Context) error {
		currentNote = ""
		selectedDateTime = ""
		selectedMonth = 0
		selectedDay = 0
		return c.Send("Напечатайте новую заметку:", &telebot.ReplyMarkup{ForceReply: true})
	})
}

// deleteHandler обработчик удалить заметку
func (w *Writer) deleteHandler() {
	w.bot.Handle("/delete", func(c telebot.Context) error {
		args := c.Args()
		if len(args) == 0 {
			return c.Send("Не указан id заметки!")
		}

		noteID, err := strconv.Atoi(args[0])
		if err != nil {
			log.Printf("failed to parse note id '%s': %v", args[0], err)
			return c.Send("Не удалось преобразовать id заметки в числовое значение!")
		}
		userID := model.UserID(c.Sender().ID)

		ctx, cancel := context.WithTimeout(context.Background(), longProcessTimeout*time.Second)
		defer cancel()

		if err = w.notes.Delete(ctx, model.NoteID(noteID), userID); err != nil {
			if errors.Is(err, model.ErrNoteNotFound) {
				return c.Send(fmt.Sprintf("Заметка '%d' не найдена", noteID))
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("context deadline exceeded while delete note %d for user '%d': %v", noteID, userID, err)
				return c.Send("Операция удаления заметки заняла слишком много времени. Попробуйте позже.")
			}
			log.Printf("failed to delete note '%d' for user '%d': %v", noteID, userID, err)
			return c.Send("Ошибка при удалении заметки. Попробуйте позже.")
		}

		return c.Send("Заметка успешно удалена")
	})
}

// getHandler обработчик получить заметку
func (w *Writer) getHandler() {
	w.bot.Handle("/get", func(c telebot.Context) error {
		args := c.Args()
		if len(args) == 0 {
			return c.Send("Не указан id заметки!")
		}

		noteID, err := strconv.Atoi(args[0])
		if err != nil {
			log.Printf("failed to parse note id '%s': %v", args[0], err)
			return c.Send("Не удалось преобразовать id заметки в числовое значение!")
		}
		userID := model.UserID(c.Sender().ID)

		ctx, cancel := context.WithTimeout(context.Background(), longProcessTimeout*time.Second)
		defer cancel()

		note, err := w.notes.Get(ctx, model.NoteID(noteID), userID)
		if err != nil {
			if errors.Is(err, model.ErrNoteNotFound) {
				return c.Send(fmt.Sprintf("Заметка '%d' не найдена", noteID))
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("context deadline exceeded while get note %d for user '%d': %v", noteID, userID, err)
				return c.Send("Операция получения заметки заняла слишком много времени. Попробуйте позже.")
			}
			log.Printf("failed to get note '%d' for user '%d': %v", noteID, userID, err)
			return c.Send("Ошибка при получении заметки. Попробуйте позже.")
		}

		messageDel := "нет"
		if note.DeletedAt != nil {
			messageDel = note.DeletedAt.Format("2006-01-02 15:04")
		}

		message := fmt.Sprintf("%s (id %d, создана: %s, напоминание: %s, удалена: %s)",
			note.Text, note.ID, note.CreatedAt.Format("2006-01-02 15:04"), note.NotifyAt.Format("2006-01-02 15:04"), messageDel)

		return c.Send(message)
	})
}

// listNoteHandler обработчик получить список заметок
func (w *Writer) listNoteHandler() {
	w.bot.Handle("/list", func(c telebot.Context) error {
		userID := model.UserID(c.Sender().ID)
		showDeleted := false

		// Проверяем наличие аргумента -a
		args := c.Args()
		if len(args) > 0 && args[0] == "-a" {
			showDeleted = true
		}

		ctx, cancel := context.WithTimeout(context.Background(), longProcessTimeout*time.Second)
		defer cancel()

		notesList, err := w.notes.List(ctx, userID, showDeleted)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Printf("context deadline exceeded while fetch notes for user '%d': %v", userID, err)
				return c.Send("Операция получения списка заметок заняла слишком много времени. Попробуйте позже.")
			}
			log.Printf("failed to fetch notes for user '%d': %v", userID, err)
			return c.Send("Ошибка при получении списка заметок. Попробуйте позже.")
		}

		if len(notesList) == 0 {
			return c.Send("Заметок нет")
		}

		var (
			response     strings.Builder
			firstDeleted = true
		)
		response.WriteString("Активные заметки:\n")
		for i, note := range notesList {
			status := ""
			if note.DeletedAt != nil {
				if firstDeleted {
					firstDeleted = false
					response.WriteString("\nУдаленные заметки:\n")
				}

				status = fmt.Sprintf(" (Удалена %s)", note.DeletedAt.Format("2006-01-02 15:04"))
			}

			response.WriteString(fmt.Sprintf("%d. %s. (id %d. Напоминание: %s)%s\n",
				i+1, note.Text, note.ID, note.NotifyAt.Format("2006-01-02 15:04"), status))
		}

		return c.Send(response.String())
	})
}

func (w *Writer) sendDays(c telebot.Context, selectedMonth time.Month) error {
	year := time.Now().Year()
	daysInMonth := time.Date(year, selectedMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
	markup := &telebot.ReplyMarkup{}
	var rows [][]telebot.InlineButton

	for day := 1; day <= daysInMonth; day++ {
		button := telebot.InlineButton{
			Unique: "select_day",
			Text:   strconv.Itoa(day),
			Data:   strconv.Itoa(day),
		}
		if day%7 == 1 {
			rows = append(rows, []telebot.InlineButton{button})
		} else {
			rows[len(rows)-1] = append(rows[len(rows)-1], button)
		}
	}

	markup.InlineKeyboard = rows
	return c.Send("Выберите день:", markup)
}

func isValidTimeFormat(input string) bool {
	if _, err := time.Parse("15", input); err == nil {
		return true
	}
	if _, err := time.Parse("15:04", input); err == nil {
		return true
	}
	return false
}

func formatTime(input string) string {
	if strings.Contains(input, ":") {
		return input
	}
	return input + ":00"
}
