---
title: 'Hugo CMS Setup Journey: Decap CMS & Sveltia CMS on GitHub Pages'
date: 2025-05-08T17:46:00
draft: false
description: This document summarizes the process of integrating a Git-based Content Management System (CMS) with the Hugo static site hosted at 0deepresearch.com on GitHub Pages. The journey involved attempting Decap CMS, switching to Sveltia CMS, troubleshooting authentication issues, and resolving content formatting problems.
---
This document summarizes the process of integrating a Git-based Content Management System (CMS) with the Hugo static site hosted at `0deepresearch.com` on GitHub Pages. The journey involved attempting Decap CMS, switching to Sveltia CMS, troubleshooting authentication issues, and resolving content formatting problems.

## 1. Initial Goal: Adding a CMS

The primary objective was to add a CMS to the existing Hugo site (`baditaflorin/0deepresearch` repository) to allow for easier content management through a web interface, rather than direct Git manipulation. The site uses Hugo with the Ananke theme and was configured with `hugo.toml`.

## 2. Attempt 1: Decap CMS Integration

**Steps:**
* Added the standard Decap CMS setup files:
    * `static/admin/index.html`: To load the Decap CMS application script.
    * `static/admin/config.yml`: To configure the connection to the GitHub repository (`baditaflorin/0deepresearch`, `main` branch) and define content collections (pages, posts).

**Problem Encountered: Authentication Failure**
When accessing `/admin/`, instead of initiating a GitHub login flow, the page redirected to `https://api.netlify.com/auth?...` with parameters like `site_id=0deepresearch.com`.
This indicated that Decap CMS was incorrectly assuming the site was hosted on Netlify and attempting to use Netlify Identity for authentication, likely confused by the custom domain (`0deepresearch.com`) being used with GitHub Pages.

**Troubleshooting:**
* Ensured `index.html` script tag was correctly placed.
* Explicitly linked `config.yml` using `<link rel="cms-config-url">`.
* Verified the `repo` path in `config.yml`.
* Commented out the `site_domain` property.
* Tested extensively with browser cache clearing and incognito windows.

The Netlify redirect persisted, suggesting a fundamental issue with direct GitHub authentication (PKCE) in the GitHub Pages + custom domain environment for Decap CMS.

## 3. Attempt 2: Switching to Sveltia CMS

**Rationale:** Based on community discussions (like GitHub issues) suggesting similar problems with Decap/Netlify CMS on non-Netlify hosting, and the recommendation that alternatives like Sveltia CMS might offer improvements or clearer paths for external authentication.

**Initial Steps:**
* Replaced the Decap CMS script tag in `static/admin/index.html` with the Sveltia CMS script tag (`https://unpkg.com/@sveltia/cms/dist/sveltia-cms.js`).
* Kept the existing `config.yml` structure (initially still configured for TOML output based on `hugo.toml`).

**Problem Encountered:** The exact same Netlify redirect occurred (`https://api.netlify.com/auth?...&site_id=0deepresearch.com...`).

**Conclusion:** This confirmed that for this specific hosting setup (GitHub Pages + Custom Domain), simply switching the CMS client-side script was insufficient. An external mechanism was needed to handle the GitHub OAuth flow correctly, bypassing any default Netlify-centric assumptions.

## 4. Solution Part 1: Implementing External Authentication

**Approach:** Followed the Sveltia CMS documentation recommendation for non-Netlify hosting by setting up their dedicated authenticator using Cloudflare Workers.

**Steps:**
1.  **Deployed Sveltia Authenticator:** Obtained the Sveltia Authenticator code and deployed it as a Cloudflare Worker (resulting URL: `https://sveltia-cms-auth.baditaflorin.workers.dev`).
2.  **Created GitHub OAuth App:** Registered a new OAuth App on GitHub, ensuring the "Authorization callback URL" was set precisely to the Cloudflare Worker's callback endpoint (`https://sveltia-cms-auth.baditaflorin.workers.dev/callback`). Obtained the Client ID and Client Secret.
3.  **Configured Cloudflare Worker:** Added the GitHub Client ID and Client Secret as encrypted environment variables (`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`) in the Cloudflare Worker settings. Also added `ALLOWED_DOMAINS` for security.
4.  **Updated `config.yml`:** Modified the `backend` section in `static/admin/config.yml` to include `base_url: https://sveltia-cms-auth.baditaflorin.workers.dev`. This explicitly tells Sveltia CMS to use the deployed worker for authentication.

**Result:** Authentication via GitHub now worked successfully, redirecting correctly through the Cloudflare Worker.

## 5. Solution Part 2: Resolving Content Formatting Issues

**Problem 1: Incorrect File Extension**
* Initially, Sveltia CMS created `.toml` files instead of the `.md` files Hugo expects for content.
* **Fix 1:** Added `extension: "md"` to the `pages` and `posts` collection definitions in `config.yml`.

**Problem 2: CMS Not Listing Existing Content / Parsing Errors**
* After setting `format:` to `toml` (to match `hugo.toml`), the CMS failed to list existing posts and showed parsing errors.
* **Diagnosis 2:** Realized existing posts still had YAML (`---`) front matter, while the CMS was now configured to expect TOML (`+++`).
* **Fix 2 (Initial):** Manually converted existing content files to use TOML front matter.

**Problem 3: Malformed TOML Output from CMS**
* Even with `format: toml` set, new files created by Sveltia CMS were still missing the `+++` delimiters and were incorrectly including the `body` content as a front matter variable (`body = "..."`). This caused Hugo to render the front matter as plain text.
* **Diagnosis 3:** Sveltia CMS's TOML generation appeared buggy or incomplete, failing to correctly structure the file with delimiters and body separation.
* **Fix 3 (The Switch):** Decided to switch the entire setup to use YAML front matter, as it's a more common default for CMS systems and potentially better supported by Sveltia CMS at this time.
    * **Hugo Config:** Renamed `hugo.toml` to `hugo.yaml` and converted its syntax.
    * **CMS Config:** Updated `static/admin/config.yml` to remove/comment out `format: toml` (letting it default to YAML) and simplified the date widget configuration. (Reflected in artifact `sveltia_config_yml_yaml`).
    * **Content Files:** Manually converted all content file front matter back to YAML (`---` delimiters, `key: value` syntax).

**Result:** Sveltia CMS now correctly creates `.md` files with valid YAML front matter (including `---` delimiters) and properly places the `body` content after the closing delimiter. Hugo renders these pages correctly.

## 6. Lessons Learned

* **Authentication on GitHub Pages (Custom Domain):** Using a Git-based CMS on GitHub Pages with a custom domain often requires an external OAuth helper service (like the Cloudflare Worker setup) to avoid conflicts with the CMS potentially defaulting to Netlify Identity assumptions. Direct PKCE authentication might not work reliably in this specific environment.
* **Front Matter Consistency:** The front matter format (`toml` vs. `yaml`) must be consistent across:
    * Hugo's main configuration file (`hugo.toml` -> expects TOML; `hugo.yaml` -> expects YAML).
    * The CMS configuration (`format:` setting in `config.yml`).
    * The actual content files (`+++` for TOML, `---` for YAML).
* **CMS Output Verification:** Always inspect the raw files created by the CMS, especially after configuration changes. Check for correct delimiters, syntax, and separation between front matter and body content.
* **Troubleshooting:** Utilize browser developer tools (console errors, network redirects) and examine raw file content to diagnose issues. Community forums and issue trackers for the specific CMS are valuable resources.
* **YAML vs. TOML:** While Hugo supports both, YAML front matter might currently be more reliably generated by Sveltia CMS than TOML front matter.

## 7. Final Working Setup

* **CMS:** Sveltia CMS (`@sveltia/cms` via unpkg).
* **Hosting:** GitHub Pages with custom domain (`0deepresearch.com`).
* **Authentication:** GitHub OAuth handled via Sveltia Authenticator deployed on Cloudflare Workers.
* **Hugo Config:** `hugo.yaml`.
* **CMS Config (`config.yml`):** Uses default (YAML) format, specifies `.md` extension. (See artifact `sveltia_config_yml_yaml`).
* **Content Files:** `.md` files with YAML front matter (`---` delimiters).
