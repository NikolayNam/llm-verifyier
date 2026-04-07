namespace CollabSphereLean

inductive Formula where
  | var : String -> Formula
  | not : Formula -> Formula
  | imp : Formula -> Formula -> Formula
deriving Repr, DecidableEq

def Formula.render : Formula -> String
  | .var name => name
  | .not value =>
      match value with
      | .var _ => "!" ++ value.render
      | .not _ => "!" ++ value.render
      | _ => "!(" ++ value.render ++ ")"
  | .imp left right =>
      let leftText :=
        match left with
        | .imp _ _ => "(" ++ left.render ++ ")"
        | _ => left.render
      let rightText :=
        match right with
        | .imp _ _ => "(" ++ right.render ++ ")"
        | _ => right.render
      leftText ++ " -> " ++ rightText

instance : ToString Formula where
  toString := Formula.render

inductive Pattern where
  | pvar : String -> Pattern
  | pnot : Pattern -> Pattern
  | pimp : Pattern -> Pattern -> Pattern
deriving Repr

def schemaA1 : Pattern :=
  .pimp (.pvar "phi") (.pimp (.pvar "psi") (.pvar "phi"))

def schemaA2 : Pattern :=
  .pimp
    (.pimp (.pvar "phi") (.pimp (.pvar "psi") (.pvar "chi")))
    (.pimp (.pimp (.pvar "phi") (.pvar "psi")) (.pimp (.pvar "phi") (.pvar "chi")))

def schemaA3 : Pattern :=
  .pimp
    (.pimp (.pnot (.pvar "psi")) (.pnot (.pvar "phi")))
    (.pimp (.pvar "phi") (.pvar "psi"))

def rulePackPattern? (name : String) : Option Pattern :=
  match name.trim with
  | "A1" => some schemaA1
  | "A2" => some schemaA2
  | "A3" => some schemaA3
  | _ => none

def envLookup : List (String × Formula) -> String -> Option Formula
  | [], _ => none
  | (key, value) :: rest, name =>
      if key = name then some value else envLookup rest name

partial def matchSchema (pattern : Pattern) (formula : Formula) (env : List (String × Formula)) :
    Option (List (String × Formula)) :=
  match pattern with
  | .pvar name =>
      match envLookup env name with
      | some bound =>
          if bound = formula then some env else none
      | none => some ((name, formula) :: env)
  | .pnot child =>
      match formula with
      | .not value => matchSchema child value env
      | _ => none
  | .pimp left right =>
      match formula with
      | .imp leftValue rightValue =>
          match matchSchema left leftValue env with
          | some nextEnv => matchSchema right rightValue nextEnv
          | none => none
      | _ => none

structure RawSourceRef where
  id : String := ""
deriving Repr

structure RawContext where
  domain : String := ""
  rulePack : String := ""
  syntaxProfile : String := ""
  generator : String := ""
deriving Repr

structure RawStep where
  kind : String := ""
  assumptionRef : Int := 0
  axiomName : String := ""
  premises : List Int := []
  formula : String := ""
  sourceRefs : List RawSourceRef := []
deriving Repr

structure RawCertificate where
  certificateVersion : String := ""
  proofId : String := ""
  goal : String := ""
  assumptions : List String := []
  context : RawContext := {}
  steps : List RawStep := []
deriving Repr

structure NormalizedCertificate where
  certificateVersion : String
  proofId : String
  goal : String
  assumptions : List String
  context : RawContext
  steps : List RawStep
deriving Repr

structure Failure where
  klass : String
  reasonCode : String
  detail : String
deriving Repr

structure Verdict where
  accepted : Bool
  klass : String
  reasonCode : String
  detail : String
deriving Repr

def acceptedVerdict : Verdict := {
  accepted := true
  klass := "accepted"
  reasonCode := "accepted"
  detail := ""
}

def failureVerdict (failure : Failure) : Verdict := {
  accepted := false
  klass := failure.klass
  reasonCode := failure.reasonCode
  detail := failure.detail
}

def schemaFailure (reasonCode detail : String) : Failure := {
  klass := "schema_error"
  reasonCode := reasonCode
  detail := detail
}

def parseFailure (reasonCode detail : String) : Failure := {
  klass := "parse_error"
  reasonCode := reasonCode
  detail := detail
}

def kernelFailure (reasonCode detail : String) : Failure := {
  klass := "kernel_validation_error"
  reasonCode := reasonCode
  detail := detail
}

def trim (value : String) : String :=
  value.trim

def isBlank (value : String) : Bool :=
  trim value = ""

def normalizeCertificate (cert : RawCertificate) : NormalizedCertificate :=
  let syntaxProfile :=
    if isBlank cert.context.syntaxProfile then
      "hilbert-prop-ascii-v1"
    else
      cert.context.syntaxProfile
  {
    certificateVersion := cert.certificateVersion
    proofId := cert.proofId
    goal := cert.goal
    assumptions := cert.assumptions
    context := { cert.context with syntaxProfile := syntaxProfile }
    steps := cert.steps
  }

def firstBlankAssumptionIndex? (items : List String) : Option Nat :=
  let rec loop (index : Nat) (remaining : List String) : Option Nat :=
    match remaining with
    | [] => none
    | head :: tail =>
        if isBlank head then
          some index
        else
          loop (index + 1) tail
  loop 0 items

def firstDuplicateAssumptionIndex? (items : List String) : Option Nat :=
  let rec loop (seen : List String) (index : Nat) (remaining : List String) : Option Nat :=
    match remaining with
    | [] => none
    | head :: tail =>
        if seen.elem head then
          some index
        else
          loop (head :: seen) (index + 1) tail
  loop [] 0 items

def validateSourceRefs (lineNo : Nat) (refs : List RawSourceRef) : Except Failure Unit :=
  let rec loop (index : Nat) (items : List RawSourceRef) : Except Failure Unit :=
    match items with
    | [] => .ok ()
    | head :: tail =>
        if isBlank head.id then
          .error <| schemaFailure "missing_source_ref_id" s!"line {lineNo}: source_refs[{index}].id is required"
        else
          loop (index + 1) tail
  loop 0 refs

def validateStep (lineNo : Nat) (step : RawStep) : Except Failure Unit := do
  if isBlank step.formula then
    throw <| schemaFailure "missing_step_formula" s!"line {lineNo}: formula is required"
  match trim step.kind with
  | "assumption" =>
      if step.assumptionRef <= 0 then
        throw <| schemaFailure "non_positive_assumption_ref" s!"line {lineNo}: assumption_ref must be positive"
      if !isBlank step.axiomName then
        throw <| schemaFailure "assumption_declares_axiom" s!"line {lineNo}: assumption step must not define axiom"
      if !step.premises.isEmpty then
        throw <| schemaFailure "assumption_declares_premises" s!"line {lineNo}: assumption step must not define premises"
  | "axiom" =>
      if isBlank step.axiomName then
        throw <| schemaFailure "missing_axiom_name" s!"line {lineNo}: axiom step requires axiom"
      if step.assumptionRef != 0 then
        throw <| schemaFailure "axiom_declares_assumption_ref" s!"line {lineNo}: axiom step must not define assumption_ref"
      if !step.premises.isEmpty then
        throw <| schemaFailure "axiom_declares_premises" s!"line {lineNo}: axiom step must not define premises"
  | "modus_ponens" =>
      if step.premises.length != 2 then
        throw <| schemaFailure "mp_wrong_premise_arity" s!"line {lineNo}: modus_ponens requires exactly 2 premises"
      if step.assumptionRef != 0 then
        throw <| schemaFailure "mp_declares_assumption_ref" s!"line {lineNo}: modus_ponens must not define assumption_ref"
      if !isBlank step.axiomName then
        throw <| schemaFailure "mp_declares_axiom" s!"line {lineNo}: modus_ponens must not define axiom"
  | other =>
      throw <| schemaFailure "unsupported_step_kind" s!"line {lineNo}: unsupported step kind \"{other}\""
  validateSourceRefs lineNo step.sourceRefs

def validateCertificate (raw : RawCertificate) : Except Failure NormalizedCertificate := do
  let cert := normalizeCertificate raw
  if trim cert.certificateVersion != "1.0.0" then
    throw <| schemaFailure "invalid_certificate_version" "certificate_version must be \"1.0.0\""
  if isBlank cert.proofId then
    throw <| schemaFailure "missing_proof_id" "proof_id is required"
  if isBlank cert.goal then
    throw <| schemaFailure "missing_goal" "goal is required"
  if isBlank cert.context.domain then
    throw <| schemaFailure "missing_context_domain" "context.domain is required"
  if isBlank cert.context.rulePack then
    throw <| schemaFailure "missing_context_rule_pack" "context.rule_pack is required"
  if isBlank cert.context.syntaxProfile then
    throw <| schemaFailure "missing_context_syntax" "context.syntax is required"
  if cert.steps.isEmpty then
    throw <| schemaFailure "missing_steps" "steps must not be empty"
  match firstBlankAssumptionIndex? cert.assumptions with
  | some index =>
      throw <| schemaFailure "blank_assumption" s!"assumptions[{index}] must not be blank"
  | none => pure ()
  match firstDuplicateAssumptionIndex? cert.assumptions with
  | some index =>
      throw <| schemaFailure "duplicate_assumption" s!"assumptions[{index}] duplicates an earlier assumption"
  | none => pure ()
  let rec loop (lineNo : Nat) (steps : List RawStep) : Except Failure Unit :=
    match steps with
    | [] => .ok ()
    | step :: tail =>
        match validateStep lineNo step with
        | .error failure => .error failure
        | .ok _ => loop (lineNo + 1) tail
  loop 1 cert.steps
  pure cert

structure ParserState where
  chars : Array Char
  pos : Nat := 0
deriving Repr

def ParserState.eof (state : ParserState) : Bool :=
  state.pos >= state.chars.size

def ParserState.peek? (state : ParserState) : Option Char :=
  state.chars[state.pos]?

def ParserState.advance (state : ParserState) (amount : Nat) : ParserState :=
  { state with pos := state.pos + amount }

def isSpaceChar (c : Char) : Bool :=
  c = ' ' || c = '\t' || c = '\n' || c = '\r'

partial def skipWhitespace (state : ParserState) : ParserState :=
  match state.peek? with
  | some c =>
      if isSpaceChar c then
        skipWhitespace (state.advance 1)
      else
        state
  | none => state

def tokenMatches (state : ParserState) (token : String) : Bool :=
  let tokenChars := token.data.toArray
  let rec loop (offset : Nat) : Bool :=
    if offset >= tokenChars.size then
      true
    else
      match state.chars[state.pos + offset]? with
      | some c =>
          match tokenChars[offset]? with
          | some expected =>
              if c = expected then
                loop (offset + 1)
              else
                false
          | none => false
      | none => false
  loop 0

def matchToken? (state : ParserState) (token : String) : Option ParserState :=
  if tokenMatches state token then
    some <| state.advance token.length
  else
    none

def identifierStart (c : Char) : Bool :=
  c.isAlpha || c = '_'

def identifierRest (c : Char) : Bool :=
  c.isAlphanum || c = '_'

partial def parseIdentifier (state : ParserState) : Except String (String × ParserState) := do
  let state := skipWhitespace state
  match state.peek? with
  | some first =>
      if !identifierStart first then
        throw s!"expected identifier at position {state.pos + 1}"
      let rec collect (pos : Nat) (acc : List Char) : String × Nat :=
        match state.chars[pos]? with
        | some c =>
            if identifierRest c then
              collect (pos + 1) (c :: acc)
            else
              (String.mk acc.reverse, pos)
        | none => (String.mk acc.reverse, pos)
      let (name, endPos) := collect (state.pos + 1) [first]
      pure (name, { state with pos := endPos })
  | none =>
      throw s!"expected identifier at position {state.pos + 1}"

mutual
  partial def parseAtom (state : ParserState) : Except String (Formula × ParserState) := do
    let state := skipWhitespace state
    if state.eof then
      throw "unexpected end of formula"
    match matchToken? state "(" with
    | some nextState =>
        let (value, afterValue) <- parseImplication nextState
        let afterValue := skipWhitespace afterValue
        match matchToken? afterValue ")" with
        | some finalState => pure (value, finalState)
        | none => throw s!"expected ')' at position {afterValue.pos + 1}"
    | none =>
        let (identifier, nextState) <- parseIdentifier state
        pure (.var identifier, nextState)

  partial def parseUnary (state : ParserState) : Except String (Formula × ParserState) := do
    let state := skipWhitespace state
    match matchToken? state "!" with
    | some nextState =>
        let (value, finalState) <- parseUnary nextState
        pure (.not value, finalState)
    | none =>
        parseAtom state

  partial def parseImplication (state : ParserState) : Except String (Formula × ParserState) := do
    let (left, nextState) <- parseUnary state
    let nextState := skipWhitespace nextState
    match matchToken? nextState "->" with
    | some afterArrow =>
        let (right, finalState) <- parseImplication afterArrow
        pure (.imp left right, finalState)
    | none => pure (left, nextState)
end

def parseFormula (input : String) : Except String Formula := do
  let trimmed := trim input
  if trimmed = "" then
    throw "formula must not be blank"
  let state : ParserState := { chars := trimmed.data.toArray, pos := 0 }
  let (formula, finalState) <- parseImplication state
  let finalState := skipWhitespace finalState
  if !finalState.eof then
    throw s!"unexpected token at position {finalState.pos + 1}"
  pure formula

def parseAssumptions (items : List String) : Except Failure (Array Formula) :=
  let rec loop (index : Nat) (remaining : List String) (acc : Array Formula) : Except Failure (Array Formula) :=
    match remaining with
    | [] => .ok acc
    | head :: tail =>
        match parseFormula head with
        | .ok formula => loop (index + 1) tail (acc.push formula)
        | .error detail =>
            .error <| parseFailure "assumption_parse_error" s!"parse assumptions[{index}]: {detail}"
  loop 0 items #[]

def parseStepFormula (lineNo : Nat) (formula : String) : Except Failure Formula :=
  match parseFormula formula with
  | .ok parsed => .ok parsed
  | .error detail => .error <| parseFailure "step_formula_parse_error" s!"line {lineNo}: parse formula: {detail}"

def parseGoalFormula (goal : String) : Except Failure Formula :=
  match parseFormula goal with
  | .ok parsed => .ok parsed
  | .error detail => .error <| parseFailure "goal_parse_error" s!"parse goal: {detail}"

def resolveRulePack (name : String) : Except Failure Unit :=
  if trim name = "classical-hilbert-v1" then
    .ok ()
  else
    .error <| kernelFailure "unsupported_rule_pack" s!"unsupported rule pack \"{name}\""

def getFormulaAt? (items : Array Formula) (lineRef : Int) : Option Formula :=
  if lineRef <= 0 then
    none
  else
    items[Int.natAbs lineRef - 1]?

partial def verifySteps (steps : List RawStep) (assumptions : Array Formula) (verified : Array Formula) (lineNo : Nat) :
    Except Failure (Array Formula) := do
  match steps with
  | [] => pure verified
  | step :: tail =>
      let formula <- parseStepFormula lineNo step.formula
      let reasonFailure? : Option Failure :=
        match trim step.kind with
        | "assumption" =>
            if step.assumptionRef <= 0 then
              some <| kernelFailure "assumption_ref_out_of_range" s!"line {lineNo}: assumption_ref {step.assumptionRef} exceeds assumptions length {assumptions.size}"
            else
              match assumptions[Int.natAbs step.assumptionRef - 1]? with
              | none =>
                  some <| kernelFailure "assumption_ref_out_of_range" s!"line {lineNo}: assumption_ref {step.assumptionRef} exceeds assumptions length {assumptions.size}"
              | some expected =>
                  if formula = expected then
                    none
                  else
                    some <| kernelFailure "assumption_mismatch" s!"line {lineNo}: assumption mismatch, expected \"{expected.render}\" but got \"{formula.render}\""
        | "axiom" =>
            match rulePackPattern? step.axiomName with
            | none =>
                some <| kernelFailure "unknown_axiom" s!"line {lineNo}: unknown axiom \"{step.axiomName}\""
            | some pattern =>
                match matchSchema pattern formula [] with
                | some _ => none
                | none =>
                    some <| kernelFailure "axiom_instance_mismatch" s!"line {lineNo}: formula \"{formula.render}\" does not match axiom {step.axiomName}"
        | "modus_ponens" =>
            match step.premises with
            | antecedentLine :: implicationLine :: _ =>
                if antecedentLine <= 0 || implicationLine <= 0 then
                  some <| kernelFailure "mp_non_positive_premise" s!"line {lineNo}: MP premises must be positive"
                else if antecedentLine >= Int.ofNat lineNo || implicationLine >= Int.ofNat lineNo then
                  some <| kernelFailure "mp_future_reference" s!"line {lineNo}: MP premises must reference previous lines only"
                else
                  match getFormulaAt? verified antecedentLine, getFormulaAt? verified implicationLine with
                  | some antecedent, some implication =>
                      match implication with
                      | .imp left right =>
                          if antecedent != left then
                            some <| kernelFailure "mp_antecedent_mismatch" s!"line {lineNo}: MP mismatch, line {antecedentLine} is \"{antecedent.render}\" but implication antecedent is \"{left.render}\""
                          else if formula != right then
                            some <| kernelFailure "mp_consequent_mismatch" s!"line {lineNo}: MP mismatch, expected consequent \"{right.render}\" but got \"{formula.render}\""
                          else
                            none
                      | _ =>
                          some <| kernelFailure "mp_non_implication_premise" s!"line {lineNo}: line {implicationLine} is not an implication, cannot use MP"
                  | _, _ =>
                      some <| kernelFailure "mp_future_reference" s!"line {lineNo}: MP premises must reference previous lines only"
            | _ =>
                some <| kernelFailure "mp_wrong_premise_arity" s!"line {lineNo}: modus_ponens requires exactly 2 premises"
        | other =>
            some <| kernelFailure "unsupported_step_kind" s!"line {lineNo}: unsupported step kind \"{other}\""
      match reasonFailure? with
      | some failure => throw failure
      | none => verifySteps tail assumptions (verified.push formula) (lineNo + 1)

def verifyCertificate (raw : RawCertificate) : Verdict :=
  match validateCertificate raw with
  | .error failure =>
      failureVerdict { failure with detail := "validate certificate json: " ++ failure.detail }
  | .ok cert =>
      match resolveRulePack cert.context.rulePack with
      | .error failure => failureVerdict failure
      | .ok _ =>
          if cert.context.syntaxProfile != "hilbert-prop-ascii-v1" then
            failureVerdict <| parseFailure "unsupported_syntax" s!"unsupported syntax \"{cert.context.syntaxProfile}\" for rule pack \"{cert.context.rulePack}\""
          else
            match parseGoalFormula cert.goal, parseAssumptions cert.assumptions with
            | .error failure, _ => failureVerdict failure
            | _, .error failure => failureVerdict failure
            | .ok goal, .ok assumptions =>
                match verifySteps cert.steps assumptions #[] 1 with
                | .error failure => failureVerdict failure
                | .ok verified =>
                    match verified[verified.size - 1]? with
                    | some last =>
                        if last = goal then
                          acceptedVerdict
                        else
                          failureVerdict <| kernelFailure "final_goal_mismatch" s!"final goal mismatch, expected \"{goal.render}\" but got \"{last.render}\""
                    | none =>
                        failureVerdict <| schemaFailure "missing_steps" "steps must not be empty"

def escapeFieldChars : List Char -> List Char
  | [] => []
  | c :: rest =>
      let escaped :=
        if c = '\\' then ['\\', '\\']
        else if c = '\t' then ['\\', 't']
        else if c = '\n' then ['\\', 'n']
        else if c = '\r' then ['\\', 'r']
        else [c]
      escaped ++ escapeFieldChars rest

def escapeField (value : String) : String :=
  String.mk <| escapeFieldChars value.data

def renderVerdictRow (caseID : String) (verdict : Verdict) : String :=
  String.intercalate "\t" [
    escapeField caseID,
    if verdict.accepted then "true" else "false",
    escapeField verdict.klass,
    escapeField verdict.reasonCode,
    escapeField verdict.detail
  ]

def renderCorpusResults (cases : List (String × RawCertificate)) : String :=
  String.intercalate "\n" <| cases.map fun (caseID, cert) =>
    renderVerdictRow caseID (verifyCertificate cert)

end CollabSphereLean

open CollabSphereLean

def corpus : List (String × RawCertificate) := [
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E09.json", { certificateVersion := "1.0.0", proofId := "V2E09", goal := "S -\u003e R", assumptions := ["R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (S -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r05/V2N07.json", { certificateVersion := "1.0.0", proofId := "cases-V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E02.json", { certificateVersion := "1.0.0", proofId := "cases-V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R-\u003eS", "S-\u003eT", "T-\u003eU"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R-\u003eS", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S-\u003eT", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T-\u003eU", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E10.json", { certificateVersion := "1.0.0", proofId := "cases-V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 4], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 5], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r06/V2N07.json", { certificateVersion := "1.0.0", proofId := "V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T -\u003e U", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e ((R -\u003e R) -\u003e R)) -\u003e ((R -\u003e (R -\u003e R)) -\u003e (R -\u003e R))", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((R -\u003e R) -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r07/V2N07.json", { certificateVersion := "1.0.0", proofId := "cases-V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R-\u003eS", "S-\u003eT", "T-\u003eU"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R-\u003eS", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S-\u003eT", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T-\u003eU", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 5], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e ((R -\u003e R) -\u003e R)) -\u003e ((R -\u003e (R -\u003e R)) -\u003e (R -\u003e R))", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((R -\u003e R) -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r08/V2N07.json", { certificateVersion := "1.0.0", proofId := "V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R-\u003eS", "S-\u003eT", "T-\u003eU"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R-\u003eS", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S-\u003eT", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T-\u003eU", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E09.json", { certificateVersion := "1.0.0", proofId := "V2E09", goal := "S -\u003e R", assumptions := ["R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (S -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R) -\u003e R", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A2", premises := [], formula := "(R -\u003e (R -\u003e R) -\u003e R) -\u003e (R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 3], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 4], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r09/V2N07.json", { certificateVersion := "1.0.0", proofId := "cases-V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E02.json", { certificateVersion := "1.0.0", proofId := "cases-V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R-\u003eS", "S-\u003eT", "T-\u003eU"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R-\u003eS", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S-\u003eT", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T-\u003eU", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 4], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 5], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "(R -\u003e ((R -\u003e R) -\u003e R)) -\u003e ((R -\u003e (R -\u003e R)) -\u003e (R -\u003e R))", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((R -\u003e R) -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-deepseek-v3-1-671b-cloud-r10/V2N07.json", { certificateVersion := "1.0.0", proofId := "cases-V2N07", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r01/V2E02.json", { certificateVersion := "1.0.0", proofId := "cases-V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r01/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T -\u003e U", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r01/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r01/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 1], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 5], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r01/V2N03.json", { certificateVersion := "1.0.0", proofId := "cases-V2N03", goal := "R", assumptions := ["R -\u003e S", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 3], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T -\u003e U", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2N03.json", { certificateVersion := "1.0.0", proofId := "cases-V2N03", goal := "R", assumptions := ["R -\u003e S", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 3], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T -\u003e U", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2N03.json", { certificateVersion := "1.0.0", proofId := "cases-V2N03", goal := "R", assumptions := ["R -\u003e S", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 3], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "T -\u003e U", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "U", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2N03.json", { certificateVersion := "1.0.0", proofId := "cases-V2N03", goal := "R", assumptions := ["R -\u003e S", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r07/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r07/V2E10.json", { certificateVersion := "1.0.0", proofId := "cases-V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r08/V2E01.json", { certificateVersion := "1.0.0", proofId := "cases-V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r08/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r08/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A2", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E01.json", { certificateVersion := "1.0.0", proofId := "V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E03.json", { certificateVersion := "1.0.0", proofId := "V2E03", goal := "T", assumptions := ["R", "R -\u003e S", "S -\u003e T"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [4, 3], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E04.json", { certificateVersion := "1.0.0", proofId := "V2E04", goal := "U", assumptions := ["R", "R -\u003e S", "S -\u003e T", "T -\u003e U"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "", sourceRefs := [] }, { kind := "assumption", assumptionRef := 4, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [5, 6], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E05.json", { certificateVersion := "1.0.0", proofId := "V2E05", goal := "R -\u003e ((S -\u003e T) -\u003e R)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((S -\u003e T) -\u003e R)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E06.json", { certificateVersion := "1.0.0", proofId := "V2E06", goal := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A2", premises := [], formula := "(R -\u003e (S -\u003e T)) -\u003e ((R -\u003e S) -\u003e (R -\u003e T))", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E07.json", { certificateVersion := "1.0.0", proofId := "V2E07", goal := "(!S -\u003e !R) -\u003e (R -\u003e S)", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A3", premises := [], formula := "(!S -\u003e !R) -\u003e (R -\u003e S)", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E08.json", { certificateVersion := "1.0.0", proofId := "V2E08", goal := "S", assumptions := ["!S -\u003e !R", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "!S -\u003e !R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A3", premises := [], formula := "(!S -\u003e !R) -\u003e (R -\u003e S)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 3], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 4], formula := "S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E09.json", { certificateVersion := "1.0.0", proofId := "V2E09", goal := "S -\u003e R", assumptions := ["R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (S -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E10.json", { certificateVersion := "1.0.0", proofId := "V2E10", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R", "S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 1], formula := "", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 4], formula := "", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E11.json", { certificateVersion := "1.0.0", proofId := "V2E11", goal := "T", assumptions := ["R -\u003e (S -\u003e T)", "R -\u003e S", "R"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 3, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }, { kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e (S -\u003e T)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 4], formula := "S -\u003e T", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [3, 5], formula := "T", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r01/V2E12.json", { certificateVersion := "1.0.0", proofId := "V2E12", goal := "R -\u003e R", assumptions := [], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A1", premises := [], formula := "R -\u003e ((R -\u003e R) -\u003e R)", sourceRefs := [] }, { kind := "axiom", assumptionRef := 0, axiomName := "A2", premises := [], formula := "(R -\u003e ((R -\u003e R) -\u003e R)) -\u003e ((R -\u003e (R -\u003e R)) -\u003e (R -\u003e R))", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [2, 3], formula := "(R -\u003e (R -\u003e R)) -\u003e (R -\u003e R)", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 4], formula := "R -\u003e R", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r02/V2E01.json", { certificateVersion := "1.0.0", proofId := "V2E01", goal := "R -\u003e S", assumptions := ["R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }] }),
  ("20260402T093635+0300_phase1-matrix-glm-5-cloud-r02/V2E02.json", { certificateVersion := "1.0.0", proofId := "V2E02", goal := "S", assumptions := ["R", "R -\u003e S"], context := { domain := "hilbert-benchmark-v1", rulePack := "classical-hilbert-v1", syntaxProfile := "hilbert-prop-ascii-v1", generator := "llm-benchmark-v1" }, steps := [{ kind := "assumption", assumptionRef := 1, axiomName := "", premises := [], formula := "R", sourceRefs := [] }, { kind := "assumption", assumptionRef := 2, axiomName := "", premises := [], formula := "R -\u003e S", sourceRefs := [] }, { kind := "modus_ponens", assumptionRef := 0, axiomName := "", premises := [1, 2], formula := "S", sourceRefs := [] }] })
]

def main : IO Unit := do
  IO.println (renderCorpusResults corpus)
