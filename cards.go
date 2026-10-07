package payriff

import "context"

type CardsService struct {
	t *transport
}

func (s *CardsService) Save(ctx context.Context, p *CardSaveParams) (*CardSaveResponse, error) {
	if p == nil {
		return nil, invalid("params are required")
	}
	if err := required(p.CustomerRef, "customerRef"); err != nil {
		return nil, err
	}
	if err := required(p.CallbackURL, "callbackURL"); err != nil {
		return nil, err
	}
	b := body{"customerRef": p.CustomerRef, "callbackUrl": p.CallbackURL}.
		set("description", p.Description).
		set("language", p.Language).
		set("metadata", p.Metadata)
	var out CardSaveResponse
	err := s.t.execute(ctx, apiRequest{
		method:  "POST",
		path:    "/api/v3/cards/save",
		headers: map[string]string{"X-Idempotency-Key": p.IdempotencyKey},
		body:    b,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CardsService) GetSave(ctx context.Context, cardSaveID string) (*CardSaveDetails, error) {
	seg, err := segment(cardSaveID, "cardSaveID")
	if err != nil {
		return nil, err
	}
	var out CardSaveDetails
	if err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/cards/save/" + seg}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CardsService) List(ctx context.Context, customerRef string) ([]SavedCard, error) {
	if _, err := segment(customerRef, "customerRef"); err != nil {
		return nil, err
	}
	out := []SavedCard{}
	err := s.t.execute(ctx, apiRequest{method: "GET", path: "/api/v3/cards/save", query: [][2]string{{"customerRef", customerRef}}}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *CardsService) Delete(ctx context.Context, cardUUID string) error {
	seg, err := segment(cardUUID, "cardUUID")
	if err != nil {
		return err
	}
	return s.t.execute(ctx, apiRequest{method: "DELETE", path: "/api/v3/cards/" + seg}, nil)
}
