## AIM

Make a utility program that allows undeleting of a file erased from an NTFS filesystem.
Think of early Norton Utilities tools.

This is purely a learning exercise.

If this works, will probably extend to other filesystems on other operating systems.

## Milestones

M1. [DONE] Make CLI skeleton
M2. [DONE] Generate Makefile so can build Windows, Linux and MacOS versions of the tools (cross compiling)
M3. Read-only block device / image abstraction
   - Open a raw volume (Windows `\\.\C:`, Linux `/dev/sdXN`, macOS `/dev/diskN`) or a disk image file.
   - Sector-aligned reads, read-only open, clear error for missing admin/root.
   - Image file support first: lets us test everywhere with no real disk and no privileges.
   - Partition table parsing (MBR/GPT) so a whole-disk image can be pointed at.

M4. NTFS boot sector (BPB) parsing
   - Bytes per sector, sectors per cluster, MFT start cluster, MFT record size, total sectors.
   - Validate "NTFS    " OEM ID and 0x55AA signature.
   - CLI: `undelete info <volume>` prints the geometry.

M5. MFT record parsing
   - Locate $MFT, read records of the right size (usually 1024 bytes).
   - Check "FILE" signature, apply the update sequence array (fixups) - easy to get wrong.
   - Header flags: in-use (0x01) and directory (0x02). In-use clear = deleted candidate.
   - Walk the attribute list: $STANDARD_INFORMATION (0x10), $FILE_NAME (0x30), $DATA (0x80), $ATTRIBUTE_LIST (0x20), end marker 0xFFFFFFFF.
   - Handle $MFT itself being fragmented (read its own $DATA runs, do not assume contiguous).

M6. List deleted files
   - Scan all MFT records, report deleted entries: name, size, timestamps, record number.
   - Rebuild full path by following the parent reference in $FILE_NAME up the tree.
     Parent may itself be deleted or reused: detect via sequence number, fall back to "<orphan>".
   - Prefer the long (Win32) name over the DOS 8.3 name.
   - CLI: `undelete list <volume>` with filters (--name glob, --min-size, --since, --dir).
   - Output formats: table, JSON.

M7. Data run decoding
   - Resident $DATA: bytes live in the MFT record, trivial case.
   - Non-resident $DATA: decode the run list (variable-width length/offset, signed relative offsets).
   - Sparse runs, multiple fragments, $ATTRIBUTE_LIST spilling attributes into extension records.
   - Unit tests with hand-built records.

M8. Recover a file (the core feature)
   - CLI: `undelete recover <volume> <record#> -o <outdir>`.
   - Read clusters from the data runs, truncate to real size, write to the output.
   - Output must be on a different volume than the source (writing to the same volume can overwrite what we want).
   - Refuse to overwrite existing output files unless --force.
   - Preserve timestamps on the recovered file where possible.

M9. Recovery confidence / overwrite detection
   - Cross-check the data runs against $Bitmap: clusters now marked allocated were probably reused.
   - Check whether the MFT record was reused (sequence number, in-use flag, different name).
   - Report per-file status: intact / partially overwritten / gone.
   - Optional: file-signature check on the first bytes (PNG, PDF, ZIP, ...) as a sanity test.

M10. Harder NTFS cases
   - Compressed files (LZNT1), sparse files, alternate data streams.
   - Encrypted (EFS): detect and report, do not try to recover.
   - Small-file edge cases: resident data, zero-length files.
   - Large volumes: stream the MFT scan, do not load it all into memory.

M11. File carving fallback
   - When the MFT record is gone, scan free clusters for known signatures (JPEG, PNG, PDF, ZIP/DOCX, ...).
   - Recovers content but not name/path. Fragmented files will be wrong; document that.
   - CLI: `undelete carve <volume> --type jpg,png`.

M12. Interactive mode (Norton-style)
   - TUI: browse the deleted list, sort/filter, mark files, recover selected.
   - Show recovery confidence inline.

M13. Test harness and CI
   - Generate small NTFS images (Linux: mkfs.ntfs + ntfs-3g to add and delete files) as fixtures.
   - Golden-file tests: delete known files, assert list and recover outputs match.
   - CI matrix building the three OS targets from M2; unit tests run on all, image tests on Linux.

M14. Safety and polish
   - Everything read-only against the source by default; a loud warning in docs about not installing or writing to the affected volume.
   - Windows: optionally read from a Volume Shadow Copy.
   - Progress reporting, cancel with Ctrl-C, sensible exit codes, --verbose logging.
   - README with usage and limitations.

M15. Other filesystems (future)
   - Introduce a `Filesystem` interface (list deleted, recover) that NTFS implements.
   - Candidates, roughly by difficulty: FAT32/exFAT (directory entries, first byte 0xE5), ext4 (journal and inode recovery), APFS/HFS+ (hard, snapshots), btrfs/XFS.
   - Do the interface refactor only when the second filesystem actually starts.

## Notes

- Order is deliberate: M3-M6 are read-only and safe, so we can learn the format with no risk to data. Writing starts at M8.
- Biggest practical limit: on SSDs with TRIM and on a busy volume, deleted data is often really gone. Set expectations in the README.
- WRITE SAFEGUARD (applies to every milestone): writes are only allowed on drive H: AND only if its volume label is exactly "UNDELETE". Implemented in `guard.go`. All file/dir creation must go through `safeCreate` / `safeMkdirAll`; never call os.Create / os.MkdirAll / os.WriteFile directly. Source volumes are opened read-only. Non-Windows builds refuse all writes until an equivalent rule is designed.
- Windows has no gcc in this environment: stay pure Go, no cgo.