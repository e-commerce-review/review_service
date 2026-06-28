package service

import "github.com/google/wire"

// ProviderSet is service providers.i
var ProviderSet = wire.NewSet(NewReviewService)
