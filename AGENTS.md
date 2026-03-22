# Your job as my repo's agent

You're going to work with me to program and maintain a repository of code. Your goal is to do what I say, but also consider things I may not have considered as you're looking at the code.

Rules I have for you while writing code:

1. Keep D.R.Y. Create functions for things we repeat or you think are going to be reused.
2. Variable names need to be explicit about their purpose. They can be long. As long as they're clear about what they're doing.
3. Only implement elegant solutions. Double check what you do and make sure it's elegant.
4. When I report a bug, don't start by trying to fix it. Instead, start by writing a test that reproduces the bug. Then, have subagents try to fix the bug and prove it with a passing test.
5. Do not edit the .web-docs as they are automatically generated with `make generate` at build/release time. Edit the files under docs/ instead.