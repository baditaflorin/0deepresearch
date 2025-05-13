# 0DeepResearch

[![Website](https://img.shields.io/badge/Website-0DeepResearch.com-blue)](https://0deepresearch.com/)

A collaborative knowledge commons for thorough research on people, organizations, and ideas. This repository powers the website at [0deepresearch.com](https://0deepresearch.com/).

## About the Project

**0DeepResearch** is built on the belief that meaningful connections start with genuine understanding. We're creating a growing repository of deeply researched profiles that doesn't disappear after a single use.

### The Vibe Briefing Concept

Our signature approach is the **Vibe Briefing**: a 10-minute micro-research sprint that reveals who someone is, what they've achieved, and where your interests might align. This pre-meeting ritual transforms networking from transactional exchanges to meaningful connections.

The foundational prompt that powers these briefings:

> "Research **[PERSON'S FULL NAME]** and compile a concise public profile: basic bio, notable achievements, and—most importantly—a list of publicly documented associates, collaborators, and friends (any context, not limited to professional work), with brief notes and sources for each connection."

### Why This Platform Exists

Every day, professionals conduct similar research on potential collaborators, clients, and partners. Yet this valuable work typically vanishes after serving its immediate purpose. **0DeepResearch** preserves and shares this collective intelligence.

## Technologies Used

- **[Hugo](https://gohugo.io/)** - Static site generator
- **[Ananke](https://github.com/theNewDynamic/gohugo-theme-ananke)** - Hugo theme
- **[Decap CMS](https://decapcms.org/)** (formerly Netlify CMS) - Content management system
- **[Sveltia Authenticator](https://github.com/sveltia/sveltia-cms)** - GitHub authentication for Decap CMS
- **GitHub Pages** - Hosting platform

## Local Development Setup

1. **Clone the repository**

```bash
git clone https://github.com/baditaflorin/0deepresearch.git
cd 0deepresearch
```

2. **Initialize submodules** (to get the Ananke theme)

```bash
git submodule update --init --recursive
```

3. **Install Hugo**

Follow the [Hugo installation guide](https://gohugo.io/getting-started/installing/) for your platform.

4. **Run local development server**

```bash
hugo server -D
```

This will start a local server at `http://localhost:1313/` that automatically refreshes when you make changes.

## Project Structure

```
.
├── archetypes/          # Templates for new content
├── assets/              # Asset files processed by Hugo pipes
├── content/             # Markdown content files
│   ├── about/           # About page
│   └── posts/           # Blog posts
├── layouts/             # Templates that define site layout
├── static/              # Static files (images, CSS, JS)
│   └── admin/           # Decap CMS configuration
└── themes/              # Hugo themes
    └── ananke/          # The Ananke theme
```

## Content Management

The site uses Decap CMS with Sveltia Authenticator for content management. The CMS is configured in `static/admin/config.yml`.

To access the CMS:
1. Go to https://0deepresearch.com/admin/
2. Log in with your GitHub account (you must be granted access to the repository)

## How to Contribute

We welcome contributions of:

1. **Original research profiles** on individuals, companies, or organizations
2. **Deep dives** into movements, concepts, or phenomena
3. **Technical guides** and methodologies for effective research

### Contribution Options for Everyone

#### Option 1: For Non-GitHub Users - Email Submission
If you don't have a GitHub account and prefer not to create one:

1. Prepare your contribution as a document (Markdown, Word, or plain text)
2. Email your contribution to [baditaflorin@gmail.com](mailto:baditaflorin@gmail.com) with:
    - Subject line: "Contribution: [Your Content Title]"
    - Brief description of your contribution
    - Your name/attribution information as you'd like it to appear
3. Our team will review your submission and handle the technical aspects of adding it to the site
4. We'll notify you when your contribution is published

#### Option 2: Using GitHub (For Technical Users)
For those familiar with GitHub or willing to learn:

1. **Create a GitHub account** if you don't have one at [github.com/signup](https://github.com/signup)
2. **Fork the repository** by clicking the "Fork" button at the top of [our GitHub page](https://github.com/baditaflorin/0deepresearch)
3. **Create your content**:
    - Navigate to the `content/posts` folder
    - Click "Add file" > "Create new file"
    - Name your file using the format: `YYYY-MM-DD-title-with-hyphens.md`
    - Add your content using Markdown format
4. **Submit a pull request**:
    - Click "Contribute" > "Open pull request"
    - Provide a title and description for your contribution
    - Click "Create pull request"
5. Wait for review and feedback from our team

### Content Formatting Guidelines

When preparing your content, please:

- Start with a clear title and introduction
- Use headings to organize information
- For research profiles, include:
    - Basic biographical information
    - Notable achievements and work
    - Professional connections and collaborations
    - Sources for all information (links, articles, etc.)
- Use Markdown formatting if possible (we provide a [simple Markdown guide](https://0deepresearch.com/markdown-guide) if needed)

### Contribution Guidelines

All contributions should:

- Use only publicly available information
- Focus on professional achievements and public associations
- Avoid speculation and unverified claims
- Include proper attribution and sources
- Respect privacy boundaries
- Be factual and well-researched

### Review Process

After submission:

1. Our review team will evaluate your contribution for quality and adherence to guidelines
2. You may receive feedback with requested revisions
3. Once approved, your contribution will be published on the site
4. Your name will be added to contributors (unless you request anonymity)

### Need Help?

If you have questions or need assistance with your contribution:
- Email [baditaflorin@gmail.com](mailto:baditaflorin@gmail.com)
- Open an issue on GitHub with your question

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the main branch.

## Contact

To get in touch about contributing or for other inquiries, please:
- Open an issue on this GitHub repository
- Reach out through the contact form on the website

---

Join us in building a community where **depth matters** and connections are built on understanding rather than chance.