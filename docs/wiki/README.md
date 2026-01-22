# Wiki Content

This folder contains the documentation content for the GitHub Wiki.

## How to Publish

GitHub Wikis are separate git repositories. To publish this content:

### Option 1: Manual Copy

1. Go to your repository on GitHub
2. Click the "Wiki" tab
3. Create pages manually and copy/paste content from each `.md` file

### Option 2: Clone Wiki Repository

```bash
# Clone the wiki repo (it's a separate git repo)
git clone https://github.com/skyhook-io/explorer.wiki.git

# Copy all markdown files
cp docs/wiki/*.md explorer.wiki/

# Commit and push
cd explorer.wiki
git add .
git commit -m "Update wiki documentation"
git push
```

## Files

| File | Description |
|------|-------------|
| `Home.md` | Wiki home page |
| `Getting-Started.md` | Installation and first run |
| `User-Guide.md` | Complete UI documentation |
| `API-Reference.md` | REST API documentation |
| `Development.md` | Contributing guide |
| `_Sidebar.md` | Wiki navigation sidebar |

## Keeping in Sync

When making documentation updates, update both:
1. `README.md` - For GitHub repository landing page
2. `docs/wiki/` - For detailed wiki documentation

Consider adding a CI step to automatically sync wiki content.
