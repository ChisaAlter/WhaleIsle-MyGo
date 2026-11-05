package pet

import (
	"errors"

	"whaleisle/internal/petdirect"
	"whaleisle/internal/store/storelocale"
)

func (q *QuickChat) sendDirect() {
	if !q.check() || q.options.Direct == nil {
		return
	}
	draft := q.draft
	ticket, ok := q.direct.Begin(draft)
	if !ok {
		q.status = storelocale.T(storelocale.KeyDesktopPetEnterUpTo2000Utf16Units)
		return
	}
	ctx, serial := q.sending.begin(q.life, petdirect.Timeout)
	if ctx == nil {
		q.direct.Cancel()
		return
	}
	q.status = storelocale.T(storelocale.KeyDesktopPetSendingToConfiguredApi)
	client, win := q.options.Direct, q.win
	go func() {
		reply, err := client.Send(ctx, ticket.Text, ticket.History)
		win.Update(func() {
			if !q.check() || !q.sending.finish(serial) {
				return
			}
			if err != nil {
				q.direct.Cancel()
				switch {
				case errors.Is(err, petdirect.ErrCredentials):
					q.status = storelocale.T(storelocale.KeyDesktopPetDesktopApiCredentialsUnavailable)
				case errors.Is(err, petdirect.ErrEndpoint):
					q.status = storelocale.T(storelocale.KeyDesktopPetApiEndpointIsInvalidOrInsecure)
				case errors.Is(err, petdirect.ErrChanged):
					q.direct.Reset()
					q.state.History = nil
					q.options.Direct.ResetBinding()
					q.status = storelocale.T(storelocale.KeyDesktopPetSettingsChangedCloseAndReopen)
				default:
					q.status = storelocale.T(storelocale.KeyDesktopPetApiResultUnconfirmedAvoidDuplicateS)
				}
				return
			}
			if !q.direct.Finish(ticket, reply) {
				return
			}
			q.state.History = append(q.state.History, quickChatRow{Role: "user", Text: ticket.Text, Seq: float64(ticket.ID * 2)}, quickChatRow{Role: "her", Text: reply, Seq: float64(ticket.ID*2 + 1)})
			if len(q.state.History) > petdirect.HistoryTurns*2 {
				q.state.History = append([]quickChatRow(nil), q.state.History[len(q.state.History)-petdirect.HistoryTurns*2:]...)
			}
			if q.draft == draft {
				q.draft = ""
			}
			q.list.ScrollToEnd()
			q.status = storelocale.T(storelocale.KeyDesktopPetTemporaryReplyReceived)
		})
	}()
}
