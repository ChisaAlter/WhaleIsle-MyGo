package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

const MessageKindQuestionReply MessageKind = "question-reply"

type QuestionReplyOption struct{ Label, Description string }
type QuestionReplyQuestion struct {
	ID, Question, Detail, Header string
	Options                      []QuestionReplyOption
}
type QuestionReplyAnswer struct {
	ID       string
	Selected []string
	Custom   string
}
type QuestionReplyView struct {
	CallID, Text string
	Time         float64
	Questions    []QuestionReplyQuestion
	Answers      []QuestionReplyAnswer
}

func classifyQuestionReply(raw json.RawMessage) (Message, bool) {
	var e struct {
		Type      string
		Seq, Time float64
		SurfaceOp struct{ Op string }
		Data      struct {
			ID      string
			Source  struct{ Kind, CallID, Outcome string }
			Content []struct{ Type, Text string }
		}
	}
	if json.Unmarshal(raw, &e) != nil || e.Type != "user/message" || e.SurfaceOp.Op != "append" || e.Data.Source.Kind != "user-question-reply" || e.Data.Source.CallID == "" || e.Data.Source.Outcome != "answered" {
		return Message{}, false
	}
	body := ""
	for _, block := range e.Data.Content {
		if block.Type == "text" {
			body = block.Text
			break
		}
	}
	id := e.Data.ID
	if id == "" {
		id = fmt.Sprintf("question-reply:%.0f", e.Seq)
	}
	return Message{ID: id, SourceMessageID: e.Data.ID, Kind: MessageKindQuestionReply, Role: "question-reply", Body: body, Seq: e.Seq, EventTime: e.Time, EventType: e.Type, Raw: append(json.RawMessage(nil), raw...)}, true
}

func QuestionReplyDetails(msg Message) QuestionReplyView {
	view := QuestionReplyView{Text: msg.Body, Time: msg.EventTime}
	var event struct {
		Data struct{ Source struct{ CallID string } }
	}
	if json.Unmarshal(msg.Raw, &event) == nil {
		view.CallID = event.Data.Source.CallID
	}
	var payload struct {
		Questions []json.RawMessage
		Answers   []json.RawMessage
	}
	if json.Unmarshal([]byte(msg.Body), &payload) != nil {
		return view
	}
	for _, raw := range payload.Questions {
		var q struct {
			ID, Question   *string
			Detail, Header string
			Options        []json.RawMessage
		}
		if json.Unmarshal(raw, &q) != nil || q.ID == nil || q.Question == nil {
			continue
		}
		question := QuestionReplyQuestion{ID: *q.ID, Question: *q.Question, Detail: q.Detail, Header: q.Header}
		for _, rawOption := range q.Options {
			var option struct {
				Label       *string
				Description string
			}
			if json.Unmarshal(rawOption, &option) == nil && option.Label != nil {
				question.Options = append(question.Options, QuestionReplyOption{*option.Label, option.Description})
			}
		}
		view.Questions = append(view.Questions, question)
	}
	for _, raw := range payload.Answers {
		var a struct {
			ID       *string
			Selected []json.RawMessage
			Custom   string
		}
		if json.Unmarshal(raw, &a) != nil || a.ID == nil || a.Selected == nil {
			continue
		}
		answer := QuestionReplyAnswer{ID: *a.ID, Custom: a.Custom}
		for _, rawSelection := range a.Selected {
			var selected string
			if json.Unmarshal(rawSelection, &selected) == nil {
				answer.Selected = append(answer.Selected, selected)
			}
		}
		view.Answers = append(view.Answers, answer)
	}
	return view
}
func (v QuestionReplyView) AnswerValues(id string) []string {
	for _, answer := range v.Answers {
		if answer.ID != id {
			continue
		}
		values := append([]string(nil), answer.Selected...)
		if custom := strings.TrimSpace(answer.Custom); custom != "" {
			values = append(values, custom)
		}
		return values
	}
	return nil
}
func (v QuestionReplyView) ClipboardText() string {
	if len(v.Questions) == 0 {
		return v.Text
	}
	blocks := []string{}
	for _, q := range v.Questions {
		heading := q.Question
		if q.Header != "" && q.Header != q.Question {
			heading = q.Header + " — " + heading
		}
		answer := "未作答 / Skipped"
		if values := v.AnswerValues(q.ID); len(values) > 0 {
			answer = "回答 / Answer: " + strings.Join(values, ", ")
		}
		blocks = append(blocks, heading+"\n"+answer)
	}
	return strings.Join(blocks, "\n\n")
}
