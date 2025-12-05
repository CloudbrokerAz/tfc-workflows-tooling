// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import "errors"

var (
	// ErrInvalidRunID indicates run ID format is invalid
	ErrInvalidRunID = errors.New("invalid run ID format")

	// ErrInvalidJustification indicates justification is missing
	ErrInvalidJustification = errors.New("justification is required")

	// ErrInvalidRunStatus indicates run is not in correct status for operation
	ErrInvalidRunStatus = errors.New("run status does not allow this operation")

	// ErrNoPolicyCheck indicates run has no policy check or task stage
	ErrNoPolicyCheck = errors.New("run has no policy evaluation")

	// ErrPolicyPending indicates policies are still being evaluated (only with NoWait=true)
	ErrPolicyPending = errors.New("policy evaluation still in progress")

	// ErrRunNotFound indicates run does not exist
	ErrRunNotFound = errors.New("run not found")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = errors.New("insufficient permissions for this operation")
)
