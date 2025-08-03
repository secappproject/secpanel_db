import 'package:flutter/material.dart';
import 'package:secpanel/theme/colors.dart';

class SearchField extends StatelessWidget {
  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  const SearchField({
    super.key,
    required this.controller,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      cursorColor: AppColors.schneiderGreen,
      style: const TextStyle(
        color: AppColors.black,
        fontSize: 12,
        fontWeight: FontWeight.w400,
      ),
      decoration: InputDecoration(
        hintText: 'No. Panel / PP / WBS / Project...',
        hintStyle: const TextStyle(
          color: AppColors.gray,
          fontSize: 12,
          fontWeight: FontWeight.w300,
        ),
        prefixIcon: const Icon(
          Icons.search,
          color: AppColors.schneiderGreen,
          size: 20,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.grayLight, width: 1),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(
            color: AppColors.schneiderGreen,
            width: 1,
          ),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.grayLight, width: 1),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        filled: true,
        fillColor: Colors.white,
      ),
      onChanged: onChanged,
    );
  }
}
