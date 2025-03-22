#!/bin/bash

# Script to convert all PDFs in pdfs/ directory to EPUBs in epubs/ directory
# Preserves metadata from original PDFs

# Create the output directory if it doesn't exist
mkdir -p epubs/

# Loop through all PDF files in the pdfs/ directory
for pdf_file in pdfs/*.pdf; do
  # Get just the filename without the path
  filename=$(basename "$pdf_file")

  # Replace the .pdf extension with .epub for the output file
  epub_file="epubs/${filename%.pdf}.epub"

  echo "Converting $pdf_file to $epub_file"

  # Convert the PDF to EPUB using Calibre's ebook-convert with recommended options
  ebook-convert "$pdf_file" "$epub_file" --no-images --enable-heuristics \
    --chapter-mark="pagebreak" --base-font-size=12 --asciiize

  # Check if conversion was successful
  if [ $? -eq 0 ]; then
    echo "Successfully converted $filename"

    # Extract metadata using exiftool (assuming it's installed)
    echo "Transferring metadata from PDF to EPUB"

    # Get title
    title=$(exiftool -s3 -Title "$pdf_file")
    if [ ! -z "$title" ]; then
      ebook-meta "$epub_file" --title="$title"
    fi

    # Get author
    author=$(exiftool -s3 -Author "$pdf_file")
    if [ -z "$author" ]; then
      author=$(exiftool -s3 -Creator "$pdf_file")
    fi
    if [ ! -z "$author" ]; then
      ebook-meta "$epub_file" --authors="$author"
    fi

    # Get publisher
    publisher=$(exiftool -s3 -Publisher "$pdf_file")
    if [ ! -z "$publisher" ]; then
      ebook-meta "$epub_file" --publisher="$publisher"
    fi

    echo "Metadata transfer completed for $filename"
  else
    echo "Error converting $filename"
  fi
done

echo "All conversions completed!"
