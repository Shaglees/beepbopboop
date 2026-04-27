from reportlab.lib.pagesizes import LETTER
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.lib import colors
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Image, PageBreak
from reportlab.lib.enums import TA_CENTER
from PIL import Image as PILImage
import os

# Best screenshot per hint (newest/most representative)
HINT_SCREENSHOTS = {
    "weather":         "/tmp/detail_screenshots/detail_weather.png",
    "brief":           "/tmp/detail_screenshots/detail_brief.png",
    "calendar":        "/tmp/detail_screenshots/detail_calendar.png",
    "deal":            "/tmp/detail_screenshots/detail_view_deal.png",
    "place":           "/tmp/detail_screenshots/detail_place2.png",
    "entertainment":   "/tmp/detail_screenshots/detail_entertainment.png",
    "movie":           "/tmp/detail_screenshots/detail_movie.png",
    "show":            "/tmp/detail_screenshots/detail_show.png",
    "restaurant":      "/tmp/detail_screenshots/detail_restaurant.png",
    "destination":     "/tmp/detail_screenshots/detail_destination.png",
    "playerSpotlight": "/tmp/detail_screenshots/detail_playerSpotlight.png",
    "album":           "/tmp/detail_screenshots/detail_album.png",
    "concert":         "/tmp/detail_screenshots/detail_concert_maybe.png",
    "gameRelease":     "/tmp/detail_screenshots/detail_game_release.png",
    "gameReview":      "/tmp/detail_screenshots/detail_gameReview_video.png",
    "science":         "/tmp/detail_screenshots/detail_science.png",
    "petSpotlight":    "/tmp/detail_screenshots/detail_pet_spotlight.png",
    "fitness":         "/tmp/detail_screenshots/detail_fitness.png",
}

PAGE_W, PAGE_H = LETTER
MARGIN = 0.6 * inch
CONTENT_W = PAGE_W - 2 * MARGIN
CONTENT_H = PAGE_H - 2 * MARGIN

def img_element(path):
    """Return a reportlab Image scaled to fit content area, preserving aspect ratio."""
    with PILImage.open(path) as im:
        w, h = im.size
    ratio = h / w
    img_w = CONTENT_W
    img_h = img_w * ratio
    max_h = CONTENT_H - 1.2 * inch  # leave room for header
    if img_h > max_h:
        img_h = max_h
        img_w = img_h / ratio
    return Image(path, width=img_w, height=img_h)

def build():
    output = "/Users/shanegleeson/Repos/beepbopboop/detail_view_report.pdf"
    doc = SimpleDocTemplate(
        output,
        pagesize=LETTER,
        leftMargin=MARGIN, rightMargin=MARGIN,
        topMargin=MARGIN, bottomMargin=MARGIN,
    )

    styles = getSampleStyleSheet()
    title_style = ParagraphStyle(
        "ReportTitle",
        parent=styles["Title"],
        fontSize=28,
        leading=36,
        textColor=colors.HexColor("#1a1a2e"),
        alignment=TA_CENTER,
        spaceAfter=16,
    )
    subtitle_style = ParagraphStyle(
        "Subtitle",
        parent=styles["Normal"],
        fontSize=14,
        textColor=colors.HexColor("#555555"),
        alignment=TA_CENTER,
    )
    hint_style = ParagraphStyle(
        "HintHeader",
        parent=styles["Heading1"],
        fontSize=22,
        leading=28,
        textColor=colors.HexColor("#1a1a2e"),
        alignment=TA_CENTER,
        spaceAfter=12,
    )
    file_style = ParagraphStyle(
        "FileName",
        parent=styles["Normal"],
        fontSize=8,
        textColor=colors.HexColor("#999999"),
        alignment=TA_CENTER,
        spaceAfter=6,
    )

    story = []

    # Title page
    story.append(Spacer(1, 2 * inch))
    story.append(Paragraph("beepbopboop", title_style))
    story.append(Paragraph("Detail View Screenshots Report", title_style))
    story.append(Spacer(1, 0.4 * inch))
    story.append(Paragraph("April 18, 2026", subtitle_style))
    story.append(Spacer(1, 0.2 * inch))
    story.append(Paragraph(f"{len(HINT_SCREENSHOTS)} hint types &bull; custom detail views", subtitle_style))
    story.append(PageBreak())

    # One page per hint
    for hint, path in HINT_SCREENSHOTS.items():
        story.append(Paragraph(hint, hint_style))
        story.append(Paragraph(os.path.basename(path), file_style))
        if os.path.exists(path):
            story.append(img_element(path))
        else:
            story.append(Paragraph(f"<i>Screenshot not found: {path}</i>", subtitle_style))
        story.append(PageBreak())

    doc.build(story)
    print(f"PDF written to {output}")

if __name__ == "__main__":
    build()
