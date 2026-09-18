package lsp

import (
	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// unimplemented satisfies protocol.Server by refusing everything.
//
// go.lsp.dev/protocol declares a 60-method interface and ships no base, so
// something has to name all 60. Embedding this one means the server
// implements a feature by writing the method for it, rather than by
// editing a dispatch table that a missing case would silently skip.
//
// Every method returns ErrMethodNotFound, which is what the protocol says
// a server should answer for a request it does not serve. The alternative,
// embedding the interface itself and leaving it nil, answers with a panic
// that takes the connection down.
//
// This file is mechanical and derived from the interface in the pinned
// version of the dependency. It changes when that version does.
type unimplemented struct{}

var _ protocol.Server = unimplemented{}

func (unimplemented) Initialize(ctx context.Context, params *protocol.InitializeParams) (result *protocol.InitializeResult, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Initialized(ctx context.Context, params *protocol.InitializedParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Shutdown(ctx context.Context) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Exit(ctx context.Context) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) LogTrace(ctx context.Context, params *protocol.LogTraceParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SetTrace(ctx context.Context, params *protocol.SetTraceParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) CodeAction(ctx context.Context, params *protocol.CodeActionParams) (result []protocol.CodeAction, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) CodeLens(ctx context.Context, params *protocol.CodeLensParams) (result []protocol.CodeLens, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) CodeLensResolve(ctx context.Context, params *protocol.CodeLens) (result *protocol.CodeLens, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) (result []protocol.ColorPresentation, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Completion(ctx context.Context, params *protocol.CompletionParams) (result *protocol.CompletionList, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (result *protocol.CompletionItem, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Declaration(ctx context.Context, params *protocol.DeclarationParams) (result []protocol.Location, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Definition(ctx context.Context, params *protocol.DefinitionParams) (result []protocol.Location, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) (result []protocol.ColorInformation, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DocumentHighlight(ctx context.Context, params *protocol.DocumentHighlightParams) (result []protocol.DocumentHighlight, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DocumentLink(ctx context.Context, params *protocol.DocumentLinkParams) (result []protocol.DocumentLink, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DocumentLinkResolve(ctx context.Context, params *protocol.DocumentLink) (result *protocol.DocumentLink, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) (result []interface{}, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (result interface{}, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) (result []protocol.FoldingRange, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Hover(ctx context.Context, params *protocol.HoverParams) (result *protocol.Hover, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Implementation(ctx context.Context, params *protocol.ImplementationParams) (result []protocol.Location, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) OnTypeFormatting(ctx context.Context, params *protocol.DocumentOnTypeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (result *protocol.Range, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) RangeFormatting(ctx context.Context, params *protocol.DocumentRangeFormattingParams) (result []protocol.TextEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) References(ctx context.Context, params *protocol.ReferenceParams) (result []protocol.Location, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Rename(ctx context.Context, params *protocol.RenameParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) (result *protocol.SignatureHelp, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) (result []protocol.SymbolInformation, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) TypeDefinition(ctx context.Context, params *protocol.TypeDefinitionParams) (result []protocol.Location, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WillSave(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WillSaveWaitUntil(ctx context.Context, params *protocol.WillSaveTextDocumentParams) (result []protocol.TextEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (result *protocol.ShowDocumentResult, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WillCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WillRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidRenameFiles(ctx context.Context, params *protocol.RenameFilesParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) WillDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (result *protocol.WorkspaceEdit, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) DidDeleteFiles(ctx context.Context, params *protocol.DeleteFilesParams) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) CodeLensRefresh(ctx context.Context) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) PrepareCallHierarchy(ctx context.Context, params *protocol.CallHierarchyPrepareParams) (result []protocol.CallHierarchyItem, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) IncomingCalls(ctx context.Context, params *protocol.CallHierarchyIncomingCallsParams) (result []protocol.CallHierarchyIncomingCall, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) OutgoingCalls(ctx context.Context, params *protocol.CallHierarchyOutgoingCallsParams) (result []protocol.CallHierarchyOutgoingCall, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (result *protocol.SemanticTokens, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (result interface{}, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SemanticTokensRange(ctx context.Context, params *protocol.SemanticTokensRangeParams) (result *protocol.SemanticTokens, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) SemanticTokensRefresh(ctx context.Context) (err error) {
	return jsonrpc2.ErrMethodNotFound
}

func (unimplemented) LinkedEditingRange(ctx context.Context, params *protocol.LinkedEditingRangeParams) (result *protocol.LinkedEditingRanges, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Moniker(ctx context.Context, params *protocol.MonikerParams) (result []protocol.Moniker, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

func (unimplemented) Request(ctx context.Context, method string, params interface{}) (result interface{}, err error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
