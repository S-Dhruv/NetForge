I am making the project to learn more about systems and see if I am interested enough to build a career out of it. Picking up a fairly simple idea and building it up from ground scratch using Go (language I have 0 knowledge in) and of course using an LLM to tutor me (so no cuck coding).

Gonna use this to see how much progress I make:
23 oct 2026 2 - 4pm: Learnt a few Go topics, designed a basic state transition for the jobs to follow and wrote a small test to see if it works. Had to rework the base idea of the project because wasn't happy. Go forward NetForge will help to research one topics better by breaking them up by planning and assigning various different jobs to be executed on the fly and reporting back with a daily or an on demand research or both.

23 oct 2026 7pm ish: Created a job struct and init behavior (since there are no explicit classes in Go). Also, created a way to determine if a certain error requires a retry and some progress made with CRUD on the database via repository.

24 oct 2026 7pm ish: Created an in memory db and crud functionality, focusing on reusing the repository interface without using the implements keyword, binding the methods to the memoryRepository struct and using a map for the storage.