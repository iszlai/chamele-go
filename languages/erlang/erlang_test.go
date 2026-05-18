package erlang_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/erlang"
)

func erlFunctions(src string) []*chamele.FunctionInfo {
	r := erlang.NewErlangReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.erl", []byte(src), r).Functions
}

func TestUnit_Erlang_Empty(t *testing.T) {
	if got := erlFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Erlang_Main(t *testing.T) {
	src := `
tail_recursive_fib(N) ->
    tail_recursive_fib(N, 0, 1, []).
lookup(_K, _Tree = ?EMPTY_NODE) ->
    {none, 'undefined'};
lookup(K, _Tree = {node, {NodeK, V, Left, Right}}) ->
    if K == NodeK -> {ok, V}
    ; K <  NodeK -> lookup(K, Left)
    ; K >  NodeK -> lookup(K, Right)
    end.
`
	got := erlFunctions(src)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].Name != "tail_recursive_fib" {
		t.Errorf("got[0].Name = %q, want tail_recursive_fib", got[0].Name)
	}
	if got[1].Name != "lookup" {
		t.Errorf("got[1].Name = %q, want lookup", got[1].Name)
	}
	if got[2].Name != "lookup" {
		t.Errorf("got[2].Name = %q, want lookup", got[2].Name)
	}
}

func TestUnit_Erlang_Nested(t *testing.T) {
	src := `
replace(Whole,Old,New) ->
    OldLen = length(Old),
    ReplaceInit = fun (Next, NewWhole) ->
              case lists:prefix(Old, [Next|NewWhole]) of
                  true ->
                      {_,Rest} = lists:split(OldLen-1, NewWhole),
                      New ++ Rest;
                  false -> [Next|NewWhole]
              end
          end,
    lists:foldr(ReplaceInit, [], Whole).
`
	got := erlFunctions(src)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "fun" {
		t.Errorf("got[0].Name = %q, want fun", got[0].Name)
	}
	if got[1].Name != "replace" {
		t.Errorf("got[1].Name = %q, want replace", got[1].Name)
	}
}

func TestUnit_Erlang_SimpleCCN(t *testing.T) {
	src := `
insert([{K, V}|Rest], Tree) ->
    insert(Rest, insert(K, V, Tree)).
`
	got := erlFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Erlang_Comments(t *testing.T) {
	src := `
%% @doc Insert a new Key into the Tree.
insert(K, V, _Tree = ?EMPTY_NODE) ->
    {node, {K, V, init(), init()}};
insert(K, V, _Tree = {node, {NodeK, NodeV, Left, Right}}) ->
    if K == NodeK -> % replace
        {node, {K, V, Left, Right}}
    ; K  < NodeK ->
        {node, {NodeK, NodeV, insert(K, V, Left), Right}}
    ; K  > NodeK ->
        {node, {NodeK, NodeV, Left, insert(K, V, Right)}}
    end.
%% @private
insert([], Tree) -> Tree;
insert([{K, V}|Rest], Tree) ->
    insert(Rest, insert(K, V, Tree)).
`
	got := erlFunctions(src)
	if len(got) != 4 {
		t.Fatalf("expected 4, got %d", len(got))
	}
	if got[0].Name != "insert" {
		t.Errorf("got[0].Name = %q, want insert", got[0].Name)
	}
	if got[1].CyclomaticComplexity != 2 {
		t.Errorf("got[1].CCN = %d, want 2", got[1].CyclomaticComplexity)
	}
}

func TestUnit_Erlang_Advanced(t *testing.T) {
	src := `
module_as_actor(E) when is_record(E, event) ->
    case lists:key_search(mfa, 1, E#event.contents) of
        {value, {mfa, {M, F, _A}}} ->
            case lists:key_search(pam_result, 1, E#event.contents) of
                {value, {pam_result, {M2, _F2, _A2}}} ->
                    {true, E#event{label = F, from = M2, to = M}};
                _ ->
                    {true, E#event{label = F, from = M, to = M}}
            end;
        _ ->
            false
    end.
`
	got := erlFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}
