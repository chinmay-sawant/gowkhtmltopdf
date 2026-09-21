package layout

// containingBlockHeight returns the current in-flow containing-block height.
// Percent heights are intentionally unresolved when the pointer is nil.
func (e *engine) containingBlockHeight() float64 {
	if e.flowCBHeight == nil {
		return -1
	}

	return *e.flowCBHeight
}

// definiteContentHeight returns the content-box height established by a
// definite height on style. The caller's containing block is used for a
// percentage height; auto height remains indefinite.
func (e *engine) definiteContentHeight(style ResolvedStyle) (float64, bool) {
	used, ok := resolveUsedHeight(&style, e.containingBlockHeight(), e)
	if !ok {
		return 0, false
	}

	content := used - style.verticalChrome(e)
	if content < 0 {
		content = 0
	}

	return content, true
}

//nolint:wsl // setting and returning the previous scope are one state transition
func (e *engine) setFlowCB(style ResolvedStyle) (*float64, float64) {
	previous := e.flowCBHeight
	parentCB := e.containingBlockHeight()
	if contentH, ok := e.definiteContentHeight(style); ok {
		e.flowCBHeight = &contentH
	} else {
		e.flowCBHeight = nil
	}

	return previous, parentCB
}
