# 09-03 Rate limits + 429 handling (Gemini free tier)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: survive exhaustion. Where: `isRateLimitOrUnavailable`, failover to Groq.
Exercise: force 429 (bad key/spam), observe failover phases.
Done when: free-tier exhaustion never blocks user.
