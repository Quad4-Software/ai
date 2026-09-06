# No AI slop rules

Upstream: github.com/realrossmanngroup/no_ai_slop_writing_rules plus MeshChatX style rules.

## Hard rules (1 through 24)

1. No emdashes. Use a period, a comma, parentheses, or restructure.
2. No unsourced statistics. Every number must be real and attributable.
3. No parenthetical clarifications in headings.
4. No intensifiers: extremely, dramatically, exceptionally, significantly, incredibly, remarkably, truly, absolutely, literally.
5. No hollow statements. Every claim ends on a concrete, verifiable detail or gets deleted.
6. No repeated talking points. Say it once.
7. Vary structure. Three consecutive sections with identical layout is a pattern; break it.
8. Reference without narrating the reference. No "as discussed above".
9. No performative urgency without a concrete consequence in the same sentence.
10. No scare quotes on normal words.
11. No filler phrases: "In today's world", "It's important to note", "When it comes to", "At the end of the day", "In the realm of", "Look no further", "Our team of experts".
12. Never start a sentence with "Whether you're".
13. Write like a researcher, not a copywriter. Anchor every sentence to something checkable.
14. No synthetic enthusiasm. No exclamation marks, no cheerleading.
15. No weasel words: "helps ensure", "may be able to", "can potentially". Commit or cut.
16. No dramatic or vague headings. A heading names what the section contains.
17. No fabricated case studies or scenarios.
18. No fabricated history or milestones. Every date must be real.
19. No fabricated attributions. Every quote traces to a real source.
20. No AI transitions: Furthermore, Moreover, Notwithstanding, That being said, At its core, In essence, It is worth noting that. Use also, and, but, however, still.
21. No AI verbs: delve, leverage, utilize, facilitate, foster, bolster, underscore, unveil, navigate (metaphorical), streamline, endeavour, ascertain, elucidate.
22. No academic tells: shed light on, pave the way for, a myriad of, a plethora of, paramount, pertaining to, prior to, subsequent to, in light of, with respect to, in terms of, the fact that.
23. Quote sources accurately. Long quotes (15+ words) get their own indented block with a one-sentence attribution.
24. No research-process narration. Silently omit what cannot be supported.

## MeshChatX additions

- No emojis anywhere in repo text or agent replies.
- No emoji arrows or decorative unicode arrows.
- No emdashes or semicolons in comments or docs.
- No backticks in code comments. Use plain words or quoted identifiers.
- Prefer fenced code blocks over inline backticks for commands, paths, and snippets.
- No TODO/FIXME comment noise.
- User-visible strings use i18n. Action feedback uses ToastUtils.

## Detection signals beyond word lists

- "It's not X. It's Y." and "It's not just X, it's Y": more than two per 500 words is a tell.
- More than 3 hedging markers (may, might, could, probably, likely, it seems) in one paragraph.
- No sentence under 8 words or over 30 words in a 500-word block: lacks burstiness.
- All paragraphs within 15% word count of each other in one section.
- More than 30% of paragraphs opening with a transition word.
- Markup artifacts: oaicite, contentReference, grok_card, attributableIndex, turn0search. Zero tolerance.

## Exclusions

Do not flag text inside fenced code blocks, direct quotations, or verbatim titles from a source.
