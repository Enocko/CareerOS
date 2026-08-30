#!/usr/bin/env python3
"""CareerOS source coverage audit - employer ATS classification."""
import json
import re
import ssl
import urllib.error
import urllib.request
from collections import Counter
from concurrent.futures import ThreadPoolExecutor, as_completed

UA = "Mozilla/5.0 (compatible; CareerOS-SourceAudit/1.0)"
ctx = ssl.create_default_context()

EMPLOYERS = {
    "Stripe": {"sector": "fintech", "url": "https://stripe.com/jobs", "slugs": ["stripe"]},
    "Datadog": {"sector": "technology", "url": "https://careers.datadoghq.com/", "slugs": ["datadog"]},
    "Cloudflare": {"sector": "technology", "url": "https://www.cloudflare.com/careers/", "slugs": ["cloudflare"]},
    "Figma": {"sector": "technology", "url": "https://www.figma.com/careers/", "slugs": ["figma"]},
    "Discord": {"sector": "technology", "url": "https://discord.com/careers", "slugs": ["discord"]},
    "Roblox": {"sector": "technology", "url": "https://careers.roblox.com/", "slugs": ["roblox"]},
    "Coinbase": {"sector": "fintech", "url": "https://www.coinbase.com/careers", "slugs": ["coinbase"]},
    "Dropbox": {"sector": "technology", "url": "https://www.dropbox.com/jobs", "slugs": ["dropbox"]},
    "Block": {"sector": "fintech", "url": "https://block.xyz/careers", "slugs": ["block"]},
    "Lyft": {"sector": "technology", "url": "https://www.lyft.com/careers", "slugs": ["lyft"]},
    "Notion": {"sector": "technology", "url": "https://www.notion.so/careers", "slugs": ["notion"]},
    "Ramp": {"sector": "fintech", "url": "https://ramp.com/careers", "slugs": ["ramp"]},
    "OpenAI": {"sector": "technology", "url": "https://openai.com/careers", "slugs": ["openai"]},
    "Plaid": {"sector": "fintech", "url": "https://plaid.com/careers/", "slugs": ["plaid"]},
    "Linear": {"sector": "technology", "url": "https://linear.app/careers", "slugs": ["linear"]},
    "Palantir": {"sector": "technology", "url": "https://www.palantir.com/careers/", "slugs": ["palantir"]},
    "Shield AI": {"sector": "defense", "url": "https://shield.ai/careers/", "slugs": ["shieldai"]},
    "Spotify": {"sector": "technology", "url": "https://www.lifeatspotify.com/jobs", "slugs": ["spotify"]},
    "Gopuff": {"sector": "technology", "url": "https://www.gopuff.com/careers", "slugs": ["gopuff"]},
    "Apple": {"sector": "technology", "url": "https://www.apple.com/careers/us/"},
    "Microsoft": {"sector": "technology", "url": "https://careers.microsoft.com/"},
    "Google": {"sector": "technology", "url": "https://www.google.com/about/careers/"},
    "Amazon": {"sector": "technology", "url": "https://www.amazon.jobs/"},
    "Meta": {"sector": "technology", "url": "https://www.metacareers.com/"},
    "NVIDIA": {"sector": "technology", "url": "https://nvidia.wd5.myworkdayjobs.com/NVIDIAExternalCareerSite"},
    "Intel": {"sector": "technology", "url": "https://jobs.intel.com/"},
    "AMD": {"sector": "technology", "url": "https://careers.amd.com/"},
    "Salesforce": {"sector": "technology", "url": "https://careers.salesforce.com/"},
    "Adobe": {"sector": "technology", "url": "https://careers.adobe.com/"},
    "ServiceNow": {"sector": "technology", "url": "https://careers.servicenow.com/"},
    "Oracle": {"sector": "technology", "url": "https://careers.oracle.com/"},
    "IBM": {"sector": "technology", "url": "https://www.ibm.com/careers"},
    "Cisco": {"sector": "technology", "url": "https://jobs.cisco.com/"},
    "Qualcomm": {"sector": "technology", "url": "https://careers.qualcomm.com/"},
    "SAP": {"sector": "enterprise", "url": "https://jobs.sap.com/"},
    "Netflix": {"sector": "technology", "url": "https://jobs.netflix.com/"},
    "Uber": {"sector": "technology", "url": "https://www.uber.com/us/en/careers/"},
    "Airbnb": {"sector": "technology", "url": "https://careers.airbnb.com/"},
    "Tesla": {"sector": "technology", "url": "https://www.tesla.com/careers"},
    "SpaceX": {"sector": "aerospace", "url": "https://www.spacex.com/careers/"},
    "Databricks": {"sector": "technology", "url": "https://www.databricks.com/company/careers"},
    "Snowflake": {"sector": "technology", "url": "https://careers.snowflake.com/"},
    "MongoDB": {"sector": "technology", "url": "https://www.mongodb.com/careers"},
    "Atlassian": {"sector": "technology", "url": "https://www.atlassian.com/company/careers"},
    "Twilio": {"sector": "technology", "url": "https://www.twilio.com/company/jobs"},
    "Okta": {"sector": "technology", "url": "https://www.okta.com/company/careers/"},
    "CrowdStrike": {"sector": "cybersecurity", "url": "https://www.crowdstrike.com/careers/"},
    "Palo Alto Networks": {"sector": "cybersecurity", "url": "https://jobs.paloaltonetworks.com/"},
    "HashiCorp": {"sector": "technology", "url": "https://www.hashicorp.com/careers"},
    "GitLab": {"sector": "technology", "url": "https://about.gitlab.com/jobs/"},
    "Confluent": {"sector": "technology", "url": "https://careers.confluent.io/"},
    "Elastic": {"sector": "technology", "url": "https://www.elastic.co/careers"},
    "Vercel": {"sector": "technology", "url": "https://vercel.com/careers"},
    "Anthropic": {"sector": "technology", "url": "https://www.anthropic.com/careers"},
    "Scale AI": {"sector": "technology", "url": "https://scale.com/careers"},
    "Anduril": {"sector": "defense", "url": "https://www.anduril.com/careers/"},
    "Benchling": {"sector": "healthtech", "url": "https://www.benchling.com/careers/"},
    "Sentry": {"sector": "technology", "url": "https://sentry.io/careers/"},
    "Amplitude": {"sector": "technology", "url": "https://amplitude.com/careers"},
    "Asana": {"sector": "technology", "url": "https://asana.com/jobs"},
    "HubSpot": {"sector": "technology", "url": "https://www.hubspot.com/careers"},
    "Zendesk": {"sector": "technology", "url": "https://www.zendesk.com/company/careers/"},
    "Intercom": {"sector": "technology", "url": "https://www.intercom.com/careers"},
    "Grammarly": {"sector": "technology", "url": "https://www.grammarly.com/careers"},
    "Canva": {"sector": "technology", "url": "https://www.canva.com/careers/"},
    "Deel": {"sector": "technology", "url": "https://www.deel.com/careers/"},
    "Goldman Sachs": {"sector": "finance", "url": "https://www.goldmansachs.com/careers/"},
    "JPMorgan Chase": {"sector": "finance", "url": "https://careers.jpmorgan.com/"},
    "Morgan Stanley": {"sector": "finance", "url": "https://morganstanley.eightfold.ai/careers"},
    "Bank of America": {"sector": "finance", "url": "https://careers.bankofamerica.com/"},
    "Citigroup": {"sector": "finance", "url": "https://jobs.citi.com/"},
    "Capital One": {"sector": "finance", "url": "https://www.capitalonecareers.com/"},
    "American Express": {"sector": "finance", "url": "https://aexp.eightfold.ai/careers"},
    "Visa": {"sector": "finance", "url": "https://careers.visa.com/"},
    "Mastercard": {"sector": "finance", "url": "https://careers.mastercard.com/"},
    "PayPal": {"sector": "fintech", "url": "https://careers.pypl.com/"},
    "Robinhood": {"sector": "fintech", "url": "https://careers.robinhood.com/"},
    "SoFi": {"sector": "fintech", "url": "https://www.sofi.com/careers/"},
    "Chime": {"sector": "fintech", "url": "https://www.chime.com/careers/"},
    "Brex": {"sector": "fintech", "url": "https://www.brex.com/careers"},
    "Affirm": {"sector": "fintech", "url": "https://www.affirm.com/careers"},
    "Jane Street": {"sector": "quant", "url": "https://www.janestreet.com/join-jane-street/"},
    "Citadel": {"sector": "quant", "url": "https://www.citadel.com/careers/"},
    "Two Sigma": {"sector": "quant", "url": "https://careers.twosigma.com/"},
    "DE Shaw": {"sector": "quant", "url": "https://www.deshaw.com/careers"},
    "Bridgewater Associates": {"sector": "quant", "url": "https://www.bridgewater.com/working-at-bridgewater"},
    "IMC Trading": {"sector": "quant", "url": "https://www.imc.com/us/careers/"},
    "Optiver": {"sector": "quant", "url": "https://optiver.com/working-at-optiver/careers/"},
    "McKinsey & Company": {"sector": "consulting", "url": "https://www.mckinsey.com/careers"},
    "BCG": {"sector": "consulting", "url": "https://careers.bcg.com/"},
    "Bain & Company": {"sector": "consulting", "url": "https://www.bain.com/careers/"},
    "Deloitte": {"sector": "consulting", "url": "https://apply.deloitte.com/"},
    "Accenture": {"sector": "consulting", "url": "https://www.accenture.com/us-en/careers"},
    "PwC": {"sector": "consulting", "url": "https://www.pwc.com/us/en/careers.html"},
    "EY": {"sector": "consulting", "url": "https://careers.ey.com/"},
    "KPMG": {"sector": "consulting", "url": "https://home.kpmg/us/en/home/careers.html"},
    "Capgemini": {"sector": "consulting", "url": "https://www.capgemini.com/careers/"},
    "Lockheed Martin": {"sector": "defense", "url": "https://www.lockheedmartinjobs.com/"},
    "Northrop Grumman": {"sector": "defense", "url": "https://www.northropgrumman.com/jobs/"},
    "Boeing": {"sector": "defense", "url": "https://jobs.boeing.com/"},
    "RTX": {"sector": "defense", "url": "https://careers.rtx.com/"},
    "General Dynamics": {"sector": "defense", "url": "https://www.gd.com/careers"},
    "L3Harris": {"sector": "defense", "url": "https://careers.l3harris.com/"},
    "UnitedHealth Group": {"sector": "healthtech", "url": "https://careers.unitedhealthgroup.com/"},
    "Epic": {"sector": "healthtech", "url": "https://careers.epic.com/"},
    "Veeva Systems": {"sector": "healthtech", "url": "https://www.veeva.com/careers/"},
    "Tempus": {"sector": "healthtech", "url": "https://www.tempus.com/careers/"},
    "DoorDash": {"sector": "technology", "url": "https://careers.doordash.com/"},
    "Instacart": {"sector": "technology", "url": "https://instacart.careers/"},
    "Walmart Global Tech": {"sector": "technology", "url": "https://careers.walmart.com/"},
    "Target": {"sector": "technology", "url": "https://corporate.target.com/careers"},
    "Snap": {"sector": "technology", "url": "https://careers.snap.com/"},
    "Pinterest": {"sector": "technology", "url": "https://www.pinterestcareers.com/"},
    "Reddit": {"sector": "technology", "url": "https://www.redditinc.com/careers"},
    "Shopify": {"sector": "technology", "url": "https://www.shopify.com/careers"},
    "Zoom": {"sector": "technology", "url": "https://careers.zoom.us/"},
    "Broadcom": {"sector": "technology", "url": "https://www.broadcom.com/company/careers"},
    "Dell Technologies": {"sector": "technology", "url": "https://jobs.dell.com/"},
    "HP Inc": {"sector": "technology", "url": "https://jobs.hp.com/"},
    "Zscaler": {"sector": "cybersecurity", "url": "https://www.zscaler.com/careers"},
    "SentinelOne": {"sector": "cybersecurity", "url": "https://www.sentinelone.com/careers/"},
    "Cohere": {"sector": "technology", "url": "https://cohere.com/careers"},
    "Monday.com": {"sector": "technology", "url": "https://monday.com/careers"},
    "Postman": {"sector": "technology", "url": "https://www.postman.com/company/careers/"},
    "Retool": {"sector": "technology", "url": "https://retool.com/careers"},
    "NASA (federal via USAJobs)": {"sector": "federal", "url": "https://www.usajobs.gov/Search/?k=NASA"},
    "DoD (federal via USAJobs)": {"sector": "federal", "url": "https://www.usajobs.gov/Search/?k=Department+of+Defense"},
}

PATTERNS = [
    ("greenhouse", r"boards\.greenhouse\.io|greenhouse\.io/embed|boards-api\.greenhouse"),
    ("ashby", r"jobs\.ashbyhq\.com|api\.ashbyhq\.com"),
    ("lever", r"jobs\.lever\.co|api\.lever\.co"),
    ("workday", r"myworkdayjobs\.com"),
    ("smartrecruiters", r"careers\.smartrecruiters\.com|jobs\.smartrecruiters\.com"),
    ("icims", r"icims\.com"),
    ("workable", r"apply\.workable\.com"),
    ("oracle", r"oraclecloud\.com/hcmUI/CandidateExperience|fa\.oraclecloud\.com"),
    ("successfactors", r"successfactors\.com|successfactors\.eu"),
    ("eightfold", r"eightfold\.ai"),
    ("phenom", r"phenompeople\.com|phenom\.com"),
    ("taleo", r"taleo\.net"),
    ("jobvite", r"jobvite\.com"),
]

SUPPORTED = {"greenhouse", "ashby", "lever", "usajobs"}


def get(url: str):
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "text/html,*/*"})
    try:
        with urllib.request.urlopen(req, timeout=15, context=ctx) as r:
            return r.geturl(), r.read(300000).decode("utf-8", "replace"), r.status
    except urllib.error.HTTPError as e:
        body = e.read(200000).decode("utf-8", "replace") if e.fp else ""
        return url, body, e.code
    except Exception as e:
        return url, str(e), 0


def probe(provider: str, slug: str) -> bool:
    urls = {
        "greenhouse": f"https://boards-api.greenhouse.io/v1/boards/{slug}/jobs?per_page=1",
        "ashby": f"https://api.ashbyhq.com/posting-api/job-board/{slug}",
        "lever": f"https://api.lever.co/v0/postings/{slug}?mode=json&limit=1",
    }
    _, body, code = get(urls[provider])
    if provider == "lever" and code != 200:
        _, body, code = get(f"https://api.eu.lever.co/v0/postings/{slug}?mode=json&limit=1")
    if code != 200:
        return False
    if provider == "greenhouse":
        return '"jobs"' in body
    if provider == "ashby":
        return '"jobs"' in body
    return body.strip().startswith("[")


def classify(name: str, meta: dict):
    if meta.get("sector") == "federal" or "usajobs.gov" in meta["url"]:
        return "usajobs", "SUPPORTED", meta["url"], "Federal postings on USAJobs.gov"

    for slug in meta.get("slugs", []):
        for provider in ("greenhouse", "ashby", "lever"):
            if probe(provider, slug):
                return provider, "SUPPORTED", meta["url"], f"Public {provider} API 200 for token '{slug}'"

    final, body, code = get(meta["url"])
    hay = (final + " " + body).lower()

    if "myworkdayjobs.com" in final.lower() or re.search(r"myworkdayjobs\.com", hay):
        return "workday", "UNSUPPORTED", final, "Workday careers URL or embedded reference"

    for provider, pattern in PATTERNS:
        if re.search(pattern, hay, re.I):
            status = "SUPPORTED" if provider in SUPPORTED else "UNSUPPORTED"
            return provider, status, final, f"HTML/URL marker for {provider} (HTTP {code})"

    custom_markers = [
        "amazon.jobs",
        "metacareers.com",
        "google.com/about/careers",
        "apple.com/careers",
        "tesla.com/careers",
        "spacex.com/careers",
        "janestreet.com",
        "deshaw.com",
        "careers.epic.com",
        "careers.microsoft.com",
        "careers.google",
    ]
    if any(m in hay for m in custom_markers):
        return "custom", "UNSUPPORTED", final, f"Company-owned careers platform (HTTP {code})"

    if code == 0:
        return "unknown", "RESEARCH NEEDED", meta["url"], f"Fetch failed: {body[:160]}"
    return "unknown", "RESEARCH NEEDED", final, f"No confident ATS marker in fetched HTML (HTTP {code})"


def main():
    results = []
    with ThreadPoolExecutor(12) as ex:
        futures = {ex.submit(classify, name, meta): name for name, meta in EMPLOYERS.items()}
        for fut in as_completed(futures):
            name = futures[fut]
            try:
                ats, status, url, evidence = fut.result()
                results.append(
                    {
                        "employer": name,
                        "sector": EMPLOYERS[name]["sector"],
                        "ats_family": ats,
                        "careeros_status": status,
                        "careers_url": EMPLOYERS[name]["url"],
                        "evidence_url": url,
                        "evidence": evidence,
                    }
                )
            except Exception as exc:
                results.append(
                    {
                        "employer": name,
                        "sector": EMPLOYERS[name]["sector"],
                        "ats_family": "unknown",
                        "careeros_status": "RESEARCH NEEDED",
                        "careers_url": EMPLOYERS[name]["url"],
                        "evidence_url": EMPLOYERS[name]["url"],
                        "evidence": str(exc),
                    }
                )

    results.sort(key=lambda r: r["employer"])
    classified = [r for r in results if r["careeros_status"] != "RESEARCH NEEDED"]
    supported = sum(1 for r in classified if r["careeros_status"] == "SUPPORTED")

    summary = {
        "total": len(results),
        "supported": supported,
        "unsupported": sum(1 for r in classified if r["careeros_status"] == "UNSUPPORTED"),
        "research_needed": sum(1 for r in results if r["careeros_status"] == "RESEARCH NEEDED"),
        "classified": len(classified),
        "coverage_pct": round(100 * supported / len(classified), 1) if classified else None,
        "ats_families": dict(Counter(r["ats_family"] for r in classified)),
        "status_counts": dict(Counter(r["careeros_status"] for r in results)),
    }

    print(json.dumps({"summary": summary, "results": results}, indent=2))


if __name__ == "__main__":
    main()
