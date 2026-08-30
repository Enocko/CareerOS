#!/usr/bin/env python3
"""Probe employer careers sources for CareerOS coverage audit. Read-only HTTP checks."""
import json
import re
import ssl
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass, asdict
from typing import Optional

TIMEOUT = 12
USER_AGENT = "CareerOS-SourceAudit/1.0 (+research; contact: careeros-audit)"

# Curated target universe (~130 employers). careers_url = official or best-known careers entry.
EMPLOYERS = [
    # Major technology
    ("Apple", "https://www.apple.com/careers/us/", "technology"),
    ("Microsoft", "https://careers.microsoft.com/", "technology"),
    ("Google", "https://www.google.com/about/careers/", "technology"),
    ("Amazon", "https://www.amazon.jobs/", "technology"),
    ("Meta", "https://www.metacareers.com/", "technology"),
    ("NVIDIA", "https://www.nvidia.com/en-us/about-nvidia/careers/", "technology"),
    ("Intel", "https://jobs.intel.com/", "technology"),
    ("AMD", "https://careers.amd.com/", "technology"),
    ("Qualcomm", "https://careers.qualcomm.com/", "technology"),
    ("Cisco", "https://jobs.cisco.com/", "technology"),
    ("IBM", "https://www.ibm.com/careers", "technology"),
    ("Oracle", "https://www.oracle.com/careers/", "technology"),
    ("Salesforce", "https://careers.salesforce.com/", "technology"),
    ("Adobe", "https://careers.adobe.com/", "technology"),
    ("ServiceNow", "https://careers.servicenow.com/", "technology"),
    ("SAP", "https://jobs.sap.com/", "enterprise"),
    ("Broadcom", "https://www.broadcom.com/company/careers", "technology"),
    ("Dell Technologies", "https://jobs.dell.com/", "technology"),
    ("HP Inc", "https://jobs.hp.com/", "technology"),
    ("Netflix", "https://jobs.netflix.com/", "technology"),
    ("Uber", "https://www.uber.com/us/en/careers/", "technology"),
    ("Airbnb", "https://careers.airbnb.com/", "technology"),
    ("Twitter/X", "https://careers.x.com/", "technology"),
    ("Snap", "https://careers.snap.com/", "technology"),
    ("Pinterest", "https://www.pinterestcareers.com/", "technology"),
    ("Reddit", "https://www.redditinc.com/careers", "technology"),
    ("Tesla", "https://www.tesla.com/careers", "technology"),
    ("SpaceX", "https://www.spacex.com/careers/", "aerospace"),
    # Already in CareerOS registry (included for completeness)
    ("Stripe", "https://stripe.com/jobs", "fintech"),
    ("Datadog", "https://careers.datadoghq.com/", "technology"),
    ("Cloudflare", "https://www.cloudflare.com/careers/", "technology"),
    ("Figma", "https://www.figma.com/careers/", "technology"),
    ("Discord", "https://discord.com/careers", "technology"),
    ("Roblox", "https://careers.roblox.com/", "technology"),
    ("Coinbase", "https://www.coinbase.com/careers", "fintech"),
    ("Dropbox", "https://www.dropbox.com/jobs", "technology"),
    ("Block", "https://block.xyz/careers", "fintech"),
    ("Lyft", "https://www.lyft.com/careers", "technology"),
    ("Notion", "https://www.notion.so/careers", "technology"),
    ("Ramp", "https://ramp.com/careers", "fintech"),
    ("OpenAI", "https://openai.com/careers", "technology"),
    ("Plaid", "https://plaid.com/careers/", "fintech"),
    ("Linear", "https://linear.app/careers", "technology"),
    ("Ashby", "https://www.ashbyhq.com/careers", "technology"),
    ("Palantir", "https://www.palantir.com/careers/", "technology"),
    ("Shield AI", "https://shield.ai/careers/", "defense"),
    ("Spotify", "https://www.lifeatspotify.com/jobs", "technology"),
    ("Gopuff", "https://www.gopuff.com/careers", "technology"),
    ("Unlimit", "https://www.unlimit.com/careers/", "fintech"),
    # High-growth / student-relevant tech
    ("Databricks", "https://www.databricks.com/company/careers", "technology"),
    ("Snowflake", "https://careers.snowflake.com/", "technology"),
    ("MongoDB", "https://www.mongodb.com/careers", "technology"),
    ("Atlassian", "https://www.atlassian.com/company/careers", "technology"),
    ("Zoom", "https://careers.zoom.us/", "technology"),
    ("Shopify", "https://www.shopify.com/careers", "technology"),
    ("Twilio", "https://www.twilio.com/company/jobs", "technology"),
    ("Okta", "https://www.okta.com/company/careers/", "technology"),
    ("CrowdStrike", "https://www.crowdstrike.com/careers/", "cybersecurity"),
    ("Palo Alto Networks", "https://jobs.paloaltonetworks.com/", "cybersecurity"),
    ("Zscaler", "https://www.zscaler.com/careers", "cybersecurity"),
    ("SentinelOne", "https://www.sentinelone.com/careers/", "cybersecurity"),
    ("Cloud Software Group", "https://www.cloudsoftwaregroup.com/careers", "technology"),
    ("HashiCorp", "https://www.hashicorp.com/careers", "technology"),
    ("GitLab", "https://about.gitlab.com/jobs/", "technology"),
    ("Confluent", "https://careers.confluent.io/", "technology"),
    ("Elastic", "https://www.elastic.co/careers", "technology"),
    ("Redis", "https://redis.com/company/careers/", "technology"),
    ("Vercel", "https://vercel.com/careers", "technology"),
    ("Retool", "https://retool.com/careers", "technology"),
    ("Deel", "https://www.deel.com/careers/", "technology"),
    ("Canva", "https://www.canva.com/careers/", "technology"),
    ("Grammarly", "https://www.grammarly.com/careers", "technology"),
    ("Asana", "https://asana.com/jobs", "technology"),
    ("Monday.com", "https://monday.com/careers", "technology"),
    ("HubSpot", "https://www.hubspot.com/careers", "technology"),
    ("Zendesk", "https://www.zendesk.com/company/careers/", "technology"),
    ("Intercom", "https://www.intercom.com/careers", "technology"),
    ("Amplitude", "https://amplitude.com/careers", "technology"),
    ("Sentry", "https://sentry.io/careers/", "technology"),
    ("Postman", "https://www.postman.com/company/careers/", "technology"),
    ("Benchling", "https://www.benchling.com/careers/", "healthtech"),
    ("Anduril", "https://www.anduril.com/careers/", "defense"),
    ("Scale AI", "https://scale.com/careers", "technology"),
    ("Anthropic", "https://www.anthropic.com/careers", "technology"),
    ("Cohere", "https://cohere.com/careers", "technology"),
    # Fintech / quant / banks
    ("Goldman Sachs", "https://www.goldmansachs.com/careers/", "finance"),
    ("JPMorgan Chase", "https://careers.jpmorgan.com/", "finance"),
    ("Morgan Stanley", "https://morganstanley.tal.net/vx/lang-en-GB/mobile-0/brand-2/candidate/jobboard/v1", "finance"),
    ("Bank of America", "https://careers.bankofamerica.com/", "finance"),
    ("Citigroup", "https://jobs.citi.com/", "finance"),
    ("Capital One", "https://www.capitalonecareers.com/", "finance"),
    ("American Express", "https://www.americanexpress.com/en-us/careers/", "finance"),
    ("Visa", "https://careers.visa.com/", "finance"),
    ("Mastercard", "https://careers.mastercard.com/", "finance"),
    ("PayPal", "https://careers.pypl.com/", "fintech"),
    ("Robinhood", "https://careers.robinhood.com/", "fintech"),
    ("SoFi", "https://www.sofi.com/careers/", "fintech"),
    ("Chime", "https://www.chime.com/careers/", "fintech"),
    ("Brex", "https://www.brex.com/careers", "fintech"),
    ("Affirm", "https://www.affirm.com/careers", "fintech"),
    ("Jane Street", "https://www.janestreet.com/join-jane-street/", "quant"),
    ("Citadel", "https://www.citadel.com/careers/", "quant"),
    ("Two Sigma", "https://careers.twosigma.com/", "quant"),
    ("DE Shaw", "https://www.deshaw.com/careers", "quant"),
    ("Bridgewater Associates", "https://www.bridgewater.com/working-at-bridgewater", "quant"),
    ("IMC Trading", "https://www.imc.com/us/careers/", "quant"),
    ("Optiver", "https://optiver.com/working-at-optiver/careers/", "quant"),
    # Consulting
    ("McKinsey & Company", "https://www.mckinsey.com/careers", "consulting"),
    ("BCG", "https://careers.bcg.com/", "consulting"),
    ("Bain & Company", "https://www.bain.com/careers/", "consulting"),
    ("Deloitte", "https://www2.deloitte.com/us/en/careers/careers.html", "consulting"),
    ("Accenture", "https://www.accenture.com/us-en/careers", "consulting"),
    ("PwC", "https://www.pwc.com/us/en/careers.html", "consulting"),
    ("EY", "https://careers.ey.com/", "consulting"),
    ("KPMG", "https://home.kpmg/us/en/home/careers.html", "consulting"),
    ("Capgemini", "https://www.capgemini.com/careers/", "consulting"),
    # Aerospace / defense
    ("Lockheed Martin", "https://www.lockheedmartinjobs.com/", "defense"),
    ("Northrop Grumman", "https://www.northropgrumman.com/jobs/", "defense"),
    ("Boeing", "https://jobs.boeing.com/", "defense"),
    ("RTX", "https://careers.rtx.com/", "defense"),
    ("General Dynamics", "https://www.gd.com/careers", "defense"),
    ("L3Harris", "https://careers.l3harris.com/", "defense"),
    # Healthcare technology
    ("UnitedHealth Group", "https://careers.unitedhealthgroup.com/", "healthtech"),
    ("Epic", "https://careers.epic.com/", "healthtech"),
    ("Veeva Systems", "https://www.veeva.com/careers/", "healthtech"),
    ("Tempus", "https://www.tempus.com/careers/", "healthtech"),
    # Consumer / logistics / other student-relevant
    ("DoorDash", "https://careers.doordash.com/", "technology"),
    ("Instacart", "https://instacart.careers/", "technology"),
    ("Walmart Global Tech", "https://careers.walmart.com/", "technology"),
    ("Target", "https://corporate.target.com/careers", "technology"),
    # Federal (USAJobs pathway)
    ("NASA (federal postings)", "https://www.usajobs.gov/Search/?k=NASA", "federal"),
    ("Department of Defense (federal)", "https://www.usajobs.gov/Search/?k=Department+of+Defense", "federal"),
]

SUPPORTED_PROVIDERS = {"greenhouse", "ashby", "lever", "usajobs"}

MARKERS = [
    ("greenhouse", [r"boards\.greenhouse\.io", r"greenhouse\.io/embed", r"boards-api\.greenhouse\.io"]),
    ("ashby", [r"jobs\.ashbyhq\.com", r"api\.ashbyhq\.com"]),
    ("lever", [r"jobs\.lever\.co", r"api\.lever\.co", r"api\.eu\.lever\.co"]),
    ("workday", [r"myworkdayjobs\.com", r"workday\.com/.*careers"]),
    ("smartrecruiters", [r"careers\.smartrecruiters\.com", r"jobs\.smartrecruiters\.com"]),
    ("icims", [r"icims\.com", r"careers-.*\.icims\.com"]),
    ("workable", [r"apply\.workable\.com", r"workable\.com"]),
    ("oracle", [r"oraclecloud\.com/hcmUI/CandidateExperience", r"fa\.oraclecloud\.com"]),
    ("successfactors", [r"successfactors\.com", r"career\d*\.successfactors"]),
    ("taleo", [r"taleo\.net"]),
    ("phenom", [r"phenompeople\.com", r"phenom\.com"]),
    ("eightfold", [r"eightfold\.ai"]),
    ("jobvite", [r"jobvite\.com"]),
    ("adp", [r"workforcenow\.adp\.com", r"recruiting\.adp\.com"]),
    ("bamboohr", [r"bamboohr\.com/jobs"]),
    ("custom", [r"amazon\.jobs", r"metacareers\.com", r"google\.com/about/careers"]),
]


def slugify(name: str) -> list[str]:
    base = re.sub(r"[^a-z0-9]+", "", name.lower())
    slugs = {base}
    # common variants
    for part in name.lower().replace("&", "and").split():
        part = re.sub(r"[^a-z0-9]", "", part)
        if part:
            slugs.add(part)
    compact = re.sub(r"[^a-z0-9]", "", name.lower())
    if compact:
        slugs.add(compact)
    # explicit known mappings
    mapping = {
        "twitterx": "twitter",
        "mckinseycompany": "mckinsey",
        "jpmorganchase": "jpmorgan",
        "bankofamerica": "bankofamerica",
        "paloaltonetworks": "paloaltonetworks",
        "general dynamics": "gd",
        "unitedhealthgroup": "unitedhealthgroup",
        "walmartglobaltech": "walmart",
        "departmentofdefensefederal": "dod",
        "nasafederalpostings": "nasa",
        "rtx": "rtx",
        "deshaw": "deshaw",
        "mondaycom": "mondaydotcom",
    }
    key = compact
    if key in mapping:
        slugs.add(mapping[key])
    return list(slugs)[:6]


def http_get(url: str, accept: str = "*/*") -> tuple[int, str, dict]:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT, "Accept": accept})
    ctx = ssl.create_default_context()
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT, context=ctx) as resp:
            body = resp.read(200_000).decode("utf-8", errors="replace")
            return resp.status, body, dict(resp.headers)
    except urllib.error.HTTPError as e:
        body = e.read(100_000).decode("utf-8", errors="replace") if e.fp else ""
        return e.code, body, dict(e.headers) if e.headers else {}
    except Exception as e:
        return 0, str(e), {}


def probe_api(provider: str, slug: str) -> bool:
    if provider == "greenhouse":
        code, body, _ = http_get(f"https://boards-api.greenhouse.io/v1/boards/{slug}/jobs?per_page=1")
        return code == 200 and '"jobs"' in body
    if provider == "ashby":
        code, body, _ = http_get(f"https://api.ashbyhq.com/posting-api/job-board/{slug}?includeCompensation=false")
        return code == 200 and '"jobs"' in body
    if provider == "lever":
        for host in ("https://api.lever.co", "https://api.eu.lever.co"):
            code, body, _ = http_get(f"{host}/v0/postings/{slug}?mode=json&limit=1")
            if code == 200 and body.strip().startswith("["):
                return True
    return False


def classify_from_html(html: str, final_url: str) -> Optional[str]:
    hay = (html + " " + final_url).lower()
    scores = {}
    for provider, patterns in MARKERS:
        for pat in patterns:
            if re.search(pat, hay, re.I):
                scores[provider] = scores.get(provider, 0) + 1
    if not scores:
        return None
    # prefer specific ATS over generic custom
    ranked = sorted(scores.items(), key=lambda x: (-x[1], x[0]))
    top = ranked[0][0]
    if top == "custom" and len(ranked) > 1:
        return ranked[1][0]
    return top


@dataclass
class Result:
    employer: str
    sector: str
    careers_url: str
    ats_family: str
    evidence: str
    careeros_status: str
    board_slug: str = ""


def audit_employer(name: str, url: str, sector: str) -> Result:
    # Federal
    if sector == "federal" or "usajobs.gov" in url:
        return Result(name, sector, url, "usajobs", "Official federal postings via USAJobs.gov", "SUPPORTED")

    # API probes for supported ATS
    for slug in slugify(name):
        for provider in ("greenhouse", "ashby", "lever"):
            if probe_api(provider, slug):
                return Result(
                    name, sector, url, provider,
                    f"Public API responded 200 for board token '{slug}' ({provider})",
                    "SUPPORTED", slug,
                )

    # Fetch careers page
    code, body, headers = http_get(url, accept="text/html")
    final_url = headers.get("Location", url) if code in (301, 302) else url
    if code in (301, 302) and headers.get("Location"):
        loc = headers["Location"]
        if loc.startswith("/"):
            from urllib.parse import urljoin
            final_url = urljoin(url, loc)
        else:
            final_url = loc
        code, body, headers = http_get(final_url, accept="text/html")

    ats = classify_from_html(body, final_url) or "unknown"
    evidence = f"HTTP {code} on {final_url}; HTML marker classification"

    # Secondary fetch if redirect chain or sparse body
    if ats == "unknown" and code == 200:
        for pat, label in [
            (r"myworkdayjobs\.com", "workday"),
            (r"greenhouse\.io", "greenhouse"),
            (r"lever\.co", "lever"),
            (r"ashbyhq\.com", "ashby"),
            (r"icims\.com", "icims"),
            (r"smartrecruiters\.com", "smartrecruiters"),
        ]:
            if re.search(pat, body, re.I):
                ats = label
                evidence = f"HTML contains {label} domain reference on {final_url}"
                break

    if ats in SUPPORTED_PROVIDERS:
        status = "SUPPORTED"
    elif ats in ("unknown",):
        status = "RESEARCH NEEDED"
    else:
        status = "UNSUPPORTED"

    return Result(name, sector, url, ats, evidence, status)


def main():
    results = []
    with ThreadPoolExecutor(max_workers=8) as ex:
        futs = {ex.submit(audit_employer, n, u, s): n for n, u, s in EMPLOYERS}
        for fut in as_completed(futs):
            try:
                results.append(fut.result())
            except Exception as e:
                results.append(Result(futs[fut], "", "", "unknown", str(e), "RESEARCH NEEDED"))

    results.sort(key=lambda r: r.employer)
    print(json.dumps([asdict(r) for r in results], indent=2))

    # summary
    from collections import Counter
    ats_counts = Counter(r.ats_family for r in results)
    status_counts = Counter(r.careeros_status for r in results)
    classified = [r for r in results if r.careeros_status != "RESEARCH NEEDED"]
    supported = sum(1 for r in classified if r.careeros_status == "SUPPORTED")
    print("\n--- SUMMARY ---", file=__import__("sys").stderr)
    print(f"Total: {len(results)}", file=__import__("sys").stderr)
    print(f"Status: {dict(status_counts)}", file=__import__("sys").stderr)
    print(f"ATS families: {dict(ats_counts)}", file=__import__("sys").stderr)
    if classified:
        print(f"Coverage: {supported}/{len(classified)} = {100*supported/len(classified):.1f}%", file=__import__("sys").stderr)


if __name__ == "__main__":
    main()
