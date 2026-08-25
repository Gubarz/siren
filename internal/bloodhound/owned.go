package bloodhound

import "context"

// IsOwned reports whether the object is owned by checking the Owned tag's
// selectors. Selector membership is what Mark/Unmark mutate, so it is
// synchronous with those calls; the graph members query can lag behind
// BloodHound's analysis and flip-flop the UI state.
func (s *Service) IsOwned(ctx context.Context, objectID string) (bool, error) {
	client, err := s.snapshot()
	if err != nil {
		return false, err
	}
	tag, err := client.Community().AssetIsolation().OwnedTag(ctx)
	if err != nil {
		return false, err
	}
	if tag.Id == nil {
		return false, nil
	}
	selectors, err := client.Community().AssetIsolation().AssetGroupTagSelectors(ctx, int32(*tag.Id))
	if err != nil {
		return false, err
	}
	for i := range selectors {
		if selectors[i].Seeds == nil {
			continue
		}
		for _, seed := range *selectors[i].Seeds {
			// Seed type 1 is an object ID per the BloodHound API contract.
			if seed.Type != nil && *seed.Type == 1 && seed.Value != nil && *seed.Value == objectID {
				return true, nil
			}
		}
	}
	return false, nil
}

// MarkOwned adds the object to BloodHound's built-in "Owned" tag, expressed
// as an object-ID selector on the tag (the SDK no-ops when already owned).
// Correlation caches are invalidated so agent chips reflect the change on
// the next correlation.
func (s *Service) MarkOwned(ctx context.Context, objectID string) error {
	client, err := s.snapshot()
	if err != nil {
		return err
	}
	if _, err := client.Community().AssetIsolation().MarkAsOwned(ctx, objectID); err != nil {
		return err
	}
	s.InvalidateCorrelation()
	return nil
}

// UnmarkOwned removes the object from the "Owned" tag; a no-op when the
// object is not owned.
func (s *Service) UnmarkOwned(ctx context.Context, objectID string) error {
	client, err := s.snapshot()
	if err != nil {
		return err
	}
	if err := client.Community().AssetIsolation().UnmarkAsOwned(ctx, objectID); err != nil {
		return err
	}
	s.InvalidateCorrelation()
	return nil
}
