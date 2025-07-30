import 'package:flutter/material.dart';
import 'package:secpanel/theme/colors.dart';

class RemarksBottomSheet extends StatelessWidget {
  final String remarks;

  const RemarksBottomSheet({super.key, required this.remarks});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        left: 20,
        right: 20,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 32,
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
          const Row(
            children: [
              Icon(Icons.notes_rounded, color: AppColors.schneiderGreen),
              SizedBox(width: 8),
              Text(
                "Busbar Remarks",
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
              ),
            ],
          ),
          const SizedBox(height: 16),
          // Container agar remarks bisa di-scroll jika teksnya sangat panjang
          Container(
            constraints: BoxConstraints(
              maxHeight: MediaQuery.of(context).size.height * 0.4,
            ),
            child: SingleChildScrollView(
              child: Text(
                remarks,
                style: const TextStyle(
                  color: AppColors.black,
                  fontSize: 14,
                  fontWeight: FontWeight.w300,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
