# ChatGPT Actions for freehire

freehire can be connected to a custom GPT through GPT Actions. ChatGPT does not
run the local `freehire` CLI; it calls the hosted HTTPS API described by
`web/static/openapi.yaml`.

## Files

- `web/static/openapi.yaml` - OpenAPI schema to import into a GPT Action.
- `web/static/.well-known/ai-plugin.json` - legacy plugin manifest for clients
  that still discover plugins through `/.well-known/ai-plugin.json`.

After deployment, the main import URL is:

```text
https://freehire.dev/openapi.yaml
```

## GPT setup

1. Create or edit a custom GPT.
2. Add an Action and import the OpenAPI schema from `https://freehire.dev/openapi.yaml`.
3. Set authentication to API key / Bearer token.
4. Create a freehire API key in the web app and paste it into the GPT Action
   authentication field.
5. Add instructions like:

```text
You help me search and track jobs in freehire.
Before using unknown filters or skills, call getJobFacets.
Use searchJobs for job search, getJob for details, getCompany for company context,
and only call saveJob, markJobApplied, updateJobTracking, unsaveJob, clearJobStage,
or deleteJobTracking after I explicitly ask you to change my job pipeline.
```

## First test prompts

```text
Find remote senior backend Go jobs in Europe. Show 10 with company, location, and URL.
```

```text
Save the first one.
```

```text
Show my applied jobs and summarize what stages they are in.
```
