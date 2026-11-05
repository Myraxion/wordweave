## Problem Statement

Wordweave 用户在收录、整理和对照阅读词条时，受到过浅文字和控件边界、长释义撑高列表、移动关系卡片窄列、表单分组不清及未保存草稿丢失的影响。键盘用户进入编辑时缺少焦点交接，辅助技术用户主动揭示答案后仍可能只能获得“收回释义”的操作名称，无法完成完整学习流程。

桌面收录与建组目前串成宽而长的单列表单，用户需要在两个独立保存的任务之间反复上下移动。保存按钮可能因多种条件禁用，但缺项解释不够就近；保存中仍能通过全局导航离开，保存结果也容易变得不明确。

本规格覆盖 2026-11-05 UI 分析报告的全部 12 项问题，以及随后访谈确认的具体布局和行为。瀑布流性能问题仅为待验证风险，不当作已发生的故障。

## Solution

保留绿色视觉风格、英文拼写的衬线字体和现有导航结构，改善可读性与键盘、屏幕阅读器使用体验。桌面收录与建组按任务分成双栏，关系组编辑按成员和关系信息分成双栏；空间不足时按相同任务顺序恢复单列。

让列表和选词候选项提供紧凑摘要，让关系卡片与详情保留全文；让手机上的成员内容自然上下排列。为草稿建立与操作影响范围一致的放弃确认，明确区分词条独立收录和关系组保存。保存附近解释当前缺项，保存中暂缓应用内跳转。完善遮挡学习，使答案在主动揭示后能被看见、读出并通过键盘完整阅读，隐藏时不泄漏。

## User Stories

1. As a notebook user, I want to retain the familiar green visual identity, spelling typography and navigation, so that I can continue using the notebook without relearning its structure.
2. As a reader, I want definitions, hints, counts and placeholders to have sufficient contrast, so that secondary information remains readable.
3. As a keyboard user, I want clearly visible control boundaries and focus indicators, so that I can identify and operate the current control.
4. As a desktop user, I want to collect entries in the left column and prepare a relation group in the right column, so that I can move between these tasks without repeatedly scrolling through a long form.
5. As an entry collector, I want each column to retain its own actions, so that I understand that collecting an entry and saving a group are separate operations.
6. As an entry collector, I want a successfully collected entry to immediately join the selected group members, so that I can continue collecting and organizing entries.
7. As a group editor, I want member selection and missing-entry collection in the left column, so that I can manage group membership in one place.
8. As a group editor, I want relation type, shared note and save actions in the right column, so that I can focus on the relationship being recorded.
9. As an entry editor, I want entry spelling and definition editing to remain a single-column task, so that a simple edit stays simple.
10. As a narrow-screen user, I want the two-column workspace to become a single column in task order, so that the controls remain usable without horizontal scrolling.
11. As a user resizing the window, I want my inputs, selections and keyboard order to remain stable, so that layout changes do not interrupt my work.
12. As a mobile reader, I want each relation-group member to show its spelling above its definition, so that long phrases are not squeezed into a narrow side column.
13. As a relation-group reader, I want full member definitions, clear group boundaries and the current-member marker, so that I can compare entries without losing context.
14. As an entry-list reader, I want ordinary definitions limited to two lines, so that one long definition does not dominate the browsing grid.
15. As a group builder, I want candidate spellings shown in full with definitions limited to two lines, so that I can recognize entries and see other candidates.
16. As a reader needing more detail, I want entry details and saved relation cards to retain full definitions, so that compact summaries do not discard my content.
17. As a group builder, I want member selection actions separated visually from relation-type fields, so that I can understand which controls belong together.
18. As a form user, I want the save area to explain the first missing requirement, so that I know how to enable saving.
19. As a form user, I want unmet conditions to look different from an active request, so that I do not mistake an incomplete form for a stalled save.
20. As an editor, I want to confirm before navigation, global search, back navigation or cancellation discards changed inputs, so that accidental actions do not erase my work.
21. As an editor who restores the original values, I want to leave without a discard warning, so that unchanged forms do not create unnecessary interruptions.
22. As an entry collector, I want a new entry to remain saved even if I abandon its pending relation group, so that independent collection is reliable.
23. As a group builder, I want a warning after collecting an entry that has joined an unsaved group, even when the group note is empty, so that I do not mistake entry collection for group creation.
24. As a user clearing entry input or cancelling missing-entry collection, I want a discard confirmation limited to that collection area, so that other group selections and notes remain intact.
25. As a user cancelling a whole page, I want one combined discard confirmation, so that several drafts do not produce repeated dialogs.
26. As a user saving content, I want application navigation and global-search navigation temporarily deferred with an explanation, so that I can understand the save result before leaving.
27. As a user whose save has finished, I want to choose navigation again myself, so that the application does not unexpectedly execute an earlier deferred jump.
28. As an editor refreshing or closing the page, I want the browser's native leave warning where supported, so that I have a chance to avoid losing unsaved work.
29. As a keyboard user opening entry editing, I want focus on the first field and restored to the trigger after cancellation, so that I can continue without restarting navigation.
30. As a keyboard user navigating pages, I want predictable main-content focus and meaningful page context, so that I can tell where navigation has taken me.
31. As a keyboard learner, I want focus alone to keep an answer hidden and Enter or Space to toggle a pinned reveal, so that merely passing a control does not expose the answer.
32. As a mouse or touch learner, I want existing hover and tap reveal behavior preserved, so that the accessibility improvements retain familiar study interactions.
33. As a screen-reader learner, I want the full answer available only after I actively reveal it and hidden again after concealment, so that I can complete recall practice without premature answer leakage.
34. As a keyboard learner reading a long answer, I want to enter, scroll, read and leave its revealed content area, so that fixed-size masks do not prevent full reading.
35. As a learner, I want spelling and definition masks to keep their fixed two-line and three-line space when revealed, so that cards do not jump or reveal answer length in advance.
36. As a learner, I want concealed answer areas excluded from keyboard navigation and accessible content, so that hidden answers remain hidden.
37. As a user encountering save or load errors, I want my draft retained and existing recovery actions available, so that I can correct the problem or retry.
38. As a frequent user, I want repeated decorative headings removed and visual scales made consistent, so that task information is easier to scan across pages.

## Implementation Decisions

### Scope and retained behavior

- Address all 12 report findings: text contrast; functional border/focus contrast; revealed-answer accessibility; missing list summaries; narrow mobile member columns; member/form grouping; overly wide single-task forms and verbose candidates; disabled-save explanation; draft loss; focus handoff; scattered design values; redundant English decorative headings.
- Preserve the entry/relation-group domain model, seven fixed relation types, current search semantics, group boundaries, current-member cues, date grouping, layout options and existing independent-save contract.
- No database migration or API contract change is expected. Use the existing entry and group operations. Do not introduce automatic group saving or turn independent entry collection into part of a group transaction.

### Layout and reading

- The desktop collection workspace has entry collection on the left and group preparation on the right. Collecting an entry saves it independently and immediately adds it to selected group members. Each task retains its own actions.
- The desktop group editor has member management and expandable missing-entry collection on the left, relation type, shared note and saving on the right. Editing an entry remains single-column.
- Fall back to a single column when both columns cannot remain comfortable to operate. The collection workspace orders entry collection before group preparation; the group editor orders members before relation information. Use the same logical keyboard order across layouts and retain state during resizing. Exact breakpoints and column proportions are implementation choices to verify in the rendered interface.
- On narrow screens, relation cards use spelling above definition per member in relation browsing, search results and entry details. Preserve complete member definitions, simple separators and the current-member text/background cue. Desktop keeps side-by-side comparison.
- Limit ordinary entry-list definitions and candidate-list definitions to two lines. Candidate spelling remains complete. Do not apply this restriction to entry details, saved relation cards or revealed fixed-size answer regions.
- Group member search, candidates and collection actions visually before the type/note fields. Maintain more space between groups than within a group.

### Visual system and focus

- Keep green identity, spelling serif typography and current navigation. Unify a minimal set of semantic text, surface, functional-border, divider, focus, spacing and typography roles rather than adopting a new component library or a large token framework.
- Normal text must meet 4.5:1 contrast, with the 3:1 exception limited to genuinely large text. Meaningful control boundaries and focus indicators must meet 3:1 against adjacent backgrounds. Distinguish functional borders from decorative dividers; do not demand functional contrast from every separator.
- Remove redundant English eyebrow headings that repeat Chinese page titles. Preserve content typography and brand recognition.
- Provide predictable keyboard focus when entering editing, cancelling editing and navigating routes. Entry editing focuses the first field; cancellation returns to its trigger. Use the existing main-content landmark for a skip-navigation path, provide meaningful page titles and avoid removing a focused trigger without handing focus to the resulting task.
- Core collection, editing, search and study workflows must be operable with keyboard and screen reader. Maintain labels, state announcements, retry and conflict-recovery actions. Do not rely on color alone for meaningful state.

### Drafts and save lifecycle

- Protect changed entry collection, entry editing, group creation and group editing drafts. Changing member sets, type or note is relevant group editing; member-search text alone is a filter, not saved group content. Evaluate actual content changes, so restoring original values removes the warning. Existing prefilled group values are the baseline, not an automatic dirty state.
- Confirm before cancellation, navigation, global-search submission or browser back discards an affected draft. Rejecting the confirmation keeps the user and draft in place. A whole-page exit uses one combined confirmation, not a dialog per draft.
- New entries remain independently saved after a group is abandoned. An independently collected entry automatically joining a pending group leaves an unsaved group draft even if the note is empty. A successful save clears only the saved task's unsaved state.
- Clearing input or cancelling missing-entry collection confirms when that local collection area has unsaved input, clears only that area and preserves group members, type and note. Successful collection may clear the collection input without a discard confirmation because it has been saved.
- Save failures retain draft content and expose existing retry/conflict recovery. A user may navigate again once the request ends, subject to ordinary unsaved-draft confirmation.
- Keep saving disabled when prerequisites are missing and explain the first missing requirement near the action: handle uncollected entry input first, then insufficient members, then missing relation type. Preserve the existing confirmation path for an edited group reduced below two members rather than silently disabling its established delete-on-save behavior. If a temporary collection editor is open, retain a clear way to finish or cancel it before group saving.
- Use waiting feedback only while a request is in progress. During a save request, defer application navigation and global-search jumps and show “正在保存，请稍候”. Do not queue or automatically replay the attempted navigation after completion; the user retries it.
- For refresh or close, use native browser leave warnings where supported. Do not promise to prevent forced close or provide automatic draft recovery. No automatic saving or draft persistence is required.

### Study interaction

- Keyboard focus alone never reveals an answer. Enter or Space toggles pinned reveal and concealment. Preserve mouse-only hover reveal, pointer-leave concealment for unpinned answers and touch tap toggling, including the existing requirement that a concealed answer under the pointer needs a new pointer entry to reveal again.
- Reveal and conceal controls need accessible names and state. The complete revealed answer must be independently readable by assistive technology; concealment must remove it from accessible content. Do not place the answer in hidden-state labels or other auxiliary names, including masked group/member headings.
- Keep fixed answer regions: spelling reserves two lines and definition three lines regardless of hidden content length. Reveal stays in the same region; overflow scrolls internally and does not resize the card.
- A revealed long-answer region is focusable and supports keyboard entry, scrolling and exit. Concealed regions are excluded from the keyboard order. If concealment happens while its answer region has focus, preserve a usable focus location on the reveal control. Use valid interaction semantics rather than nesting controls inside a button.
- Preserve independent detail entry points, shared display-mode preference, reset of pinned reveals on mode changes/page exit, initially collapsed relation notes in study modes and omission of empty relation notes. Do not add visible “显示拼写/释义” text to masks.

### Delivery approach

- Work in batches: high-impact contrast/accessibility/list-summary corrections, task layout and form feedback, draft and navigation behavior, then consistent scales and visual cleanup. The specification is one deliverable; separate issues or new architectural layers are not required.
- Prefer small changes within the existing application shell, collection/group editor, browsing cards, study-reveal control and shared styling. Share only the small amount of draft/busy coordination needed across the shell and forms.
- The suspected masonry layout read/write problem must be measured before claiming a performance regression. Do not introduce virtualization or a new layout engine based solely on static suspicion.

## Testing Decisions

- **User-confirmed seam:** use the existing browser-level page/workflow boundary as the primary test seam. Reuse Playwright and the project's existing browser-test style; supplement it with a small real-Go-service persistence check and manual keyboard/screen-reader verification. Avoid adding per-helper or CSS-token tests that simply mirror implementation.
- Existing prior art includes browser checks of date-grouped browsing with intercepted API fixtures, HTTP-level application tests against isolated real SQLite databases and executable lifecycle tests. Extend these conventions instead of introducing a separate UI test framework.
- Good tests observe user-visible text, accessible roles/names/states, focus, navigation, saved data and rendered layout. Do not assert private refs, component decomposition, literal token names or exact CSS declarations.
- Exercise the collection workspace, group editor, entry editor, application navigation/global search, browsing cards and reveal controls through the same browser boundary. Use labels and roles where available; use layout measurements only for actual layout requirements.
- Verify desktop task separation and narrow-screen task order with short content, long English phrases, multiline/empty definitions and long group notes. Check representative widths including 320, 390, 651, 768 and 1440 CSS pixels. Assert no horizontal page overflow and no draft loss during resizing; do not freeze a particular breakpoint as a test implementation detail.
- Verify two-line list/candidate summaries while detail and relation-card content remains available in full. Check logical group spacing visually at desktop and mobile sizes rather than snapshotting every spacing value.
- Verify blank/unchanged forms leave without warnings; changed spelling, definition, member set, type and note trigger protection; restoring originals removes it; declining confirmation retains content. Cover cancellation, sidebar/return navigation, global search and browser back, including switching away from inline forms.
- Verify local clear/cancel affects only local collection inputs, whole-page exit prompts once, and successful collection leaves a pending group draft. With the real Go service and an isolated data directory, collect an entry, abandon group creation, reload and verify the entry remains while no group was created. Also verify a successful group save is observable after reload. Mock fixtures alone cannot establish persistence.
- Control request completion at the browser's network boundary to verify save-in-progress navigation is deferred with the explanation, no deferred jump automatically replays after success or failure, drafts survive failure, and successful saves do not prompt as if their task were still unsaved.
- Verify reveal controls using Enter and Space, focus without reveal, reveal/conceal accessibility content, long-answer keyboard scrolling/exit, fixed geometry, mode/route reset and independent details navigation. Preserve mouse hover/pin/conceal behavior. A mobile viewport proves layout only; synthesize touch where supported and state explicitly if real-touch behavior remains untested.
- Inspect normal, hover, focused, selected and error states for relevant text/control contrast. Disabled controls and decorative dividers have different applicability; do not treat all controls below 44px as automatic WCAG AA failures.
- Manually complete core collection, editing, search and study flows with a keyboard and a screen reader, documenting browser/assistive-technology versions and remaining limitations. Check reading order, answer leakage, full revealed answers, focus restoration and status messages. Automated AX checks supplement rather than replace this verification.
- Run existing relevant browser regressions, frontend build/type checking and existing backend tests appropriate to preserved API behavior. No database test expansion is needed solely for CSS or layout edits.
- Prior audit build/type checking passed, but its automated browser run stopped during browser startup before assertions. Resolve or clearly report that environment issue; neither mark blocked assertions as product failures nor claim they passed. The static detector's empty results are not an accessibility pass.

## Out of Scope

- Replacing visual identity, navigation architecture, the relation-group domain model, storage technology or public API contracts.
- New relation types, separate group names, automatic dictionary lookup, pronunciation, spaced repetition, quizzes, multiple users/notebooks, import/export or rich-text editing.
- A multi-step collection wizard, automatic saving, persistent draft recovery or undo infrastructure.
- Dark mode, a new component library or a comprehensive design-system rewrite.
- New browsing-state persistence, changes to search matching, sorting/date semantics or relation inference.
- A blanket large-data performance rewrite or mandatory virtualization without measurements.
- Guarantees against forced browser/process termination, universal assistive-technology certification or treating viewport emulation as physical-device testing.

## Further Notes

- This specification synthesizes the UI audit and all confirmed interview answers through 2026-11-05, including the user's explicit choice of desktop two-column operation over the report's initial single-column suggestion. In a conflict, these confirmed decisions take precedence over the initial design recommendation.
- The audit recorded four P1, seven P2 and one P3 findings. Its numerical scores are prioritization aids, not acceptance criteria or a request to redesign the application.
- Preserve simple, maintainable implementation. Existing domain decisions remain applicable; no new domain term or irreversible architectural trade-off currently requires a glossary addition or ADR.
- Scope and behavior are settled. Pixel-level spacing, exact breakpoints, accessible DOM arrangement and small internal interfaces remain implementation choices constrained by the observable behavior and verification above.
