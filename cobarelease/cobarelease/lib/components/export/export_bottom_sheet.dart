import 'package:flutter/material.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class PreviewBottomSheet extends StatefulWidget {
  final Company currentUser;

  const PreviewBottomSheet({super.key, required this.currentUser});

  @override
  State<PreviewBottomSheet> createState() => _PreviewBottomSheetState();
}

class _PreviewBottomSheetState extends State<PreviewBottomSheet> {
  // --- [PERUBAHAN] State disederhanakan ---
  bool _exportPanelData = true;
  bool _exportUserData = true;
  String _selectedFormat = 'Excel';
  // --- [AKHIR PERUBAHAN] ---

  Widget _buildSectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(top: 24.0, bottom: 12.0),
      child: Text(
        title,
        style: const TextStyle(
          fontFamily: 'Lexend',
          fontSize: 14,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }

  // --- [PERUBAHAN] Widget baru untuk opsi on/off ---
  Widget _buildToggleOption({
    required String label,
    required bool isSelected,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        margin: const EdgeInsets.only(right: 8, bottom: 12),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(
            color: isSelected ? AppColors.schneiderGreen : AppColors.grayLight,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              isSelected ? Icons.check_box : Icons.check_box_outline_blank,
              color: isSelected ? AppColors.schneiderGreen : AppColors.gray,
              size: 20,
            ),
            const SizedBox(width: 8),
            Text(
              label,
              style: const TextStyle(fontWeight: FontWeight.w400, fontSize: 12),
            ),
          ],
        ),
      ),
    );
  }
  // --- [AKHIR PERUBAHAN] ---

  Widget _buildSingleSelectOption(String format) {
    final bool isSelected = _selectedFormat == format;
    return GestureDetector(
      onTap: () {
        setState(() {
          _selectedFormat = format;
        });
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        margin: const EdgeInsets.only(right: 8, bottom: 12),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(
            color: isSelected ? AppColors.schneiderGreen : AppColors.grayLight,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          format,
          style: const TextStyle(fontWeight: FontWeight.w400, fontSize: 12),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    // --- [PERUBAHAN] Kondisi untuk menonaktifkan tombol ekspor ---
    final bool isAnyDataSelected = _exportPanelData || _exportUserData;

    return Padding(
      padding: EdgeInsets.only(
        left: 20,
        right: 20,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              height: 5,
              width: 40,
              decoration: BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.circular(100),
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            "Ekspor Data",
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.w400),
          ),
          Flexible(
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // --- [PERUBAHAN] Opsi ekspor yang baru ---
                  _buildSectionTitle("Pilih Data untuk Diekspor"),
                  Wrap(
                    children: [
                      _buildToggleOption(
                        label: "Data Panel & Relasi",
                        isSelected: _exportPanelData,
                        onTap: () {
                          setState(() => _exportPanelData = !_exportPanelData);
                        },
                      ),
                      _buildToggleOption(
                        label: "Data User & Relasi",
                        isSelected: _exportUserData,
                        onTap: () {
                          setState(() => _exportUserData = !_exportUserData);
                        },
                      ),
                    ],
                  ),
                  // --- [AKHIR PERUBAHAN] ---
                  _buildSectionTitle("Pilih Format File"),
                  Wrap(
                    children: ['Excel', 'JSON']
                        .map((format) => _buildSingleSelectOption(format))
                        .toList(),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    side: const BorderSide(color: AppColors.schneiderGreen),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text(
                    'Batal',
                    style: TextStyle(
                      color: AppColors.schneiderGreen,
                      fontSize: 12,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    backgroundColor: AppColors.schneiderGreen,
                    foregroundColor: Colors.white,
                    disabledBackgroundColor: AppColors.grayNeutral,
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  // --- [PERUBAHAN] Logika onPressed disesuaikan ---
                  onPressed: !isAnyDataSelected
                      ? null
                      : () {
                          Navigator.of(context).pop({
                            'exportPanel': _exportPanelData,
                            'exportUser': _exportUserData,
                            'format': _selectedFormat,
                          });
                        },
                  child: const Text('Ekspor', style: TextStyle(fontSize: 12)),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
