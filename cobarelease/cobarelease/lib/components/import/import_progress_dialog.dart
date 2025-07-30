// lib/components/import_progress_dialog.dart

import 'package:flutter/material.dart';
import 'package:secpanel/theme/colors.dart';

class ImportProgressDialog extends StatelessWidget {
  final ValueNotifier<double> progress;
  final ValueNotifier<String> status;

  const ImportProgressDialog({
    super.key,
    required this.progress,
    required this.status,
  });

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      backgroundColor: AppColors.white,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Text(
            "Mengimpor Data...",
            style: TextStyle(fontWeight: FontWeight.w500, fontSize: 16),
          ),
          const SizedBox(height: 20),
          ValueListenableBuilder<double>(
            valueListenable: progress,
            builder: (context, value, _) {
              return LinearProgressIndicator(
                value: value,
                backgroundColor: AppColors.grayLight,
                color: AppColors.schneiderGreen,
                minHeight: 6,
                borderRadius: BorderRadius.circular(100),
              );
            },
          ),
          const SizedBox(height: 12),
          ValueListenableBuilder<String>(
            valueListenable: status,
            builder: (context, value, _) {
              return Text(
                value,
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 12, color: AppColors.gray),
              );
            },
          ),
        ],
      ),
    );
  }
}
