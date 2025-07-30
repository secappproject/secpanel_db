import 'package:flutter/material.dart';
import 'package:secpanel/theme/colors.dart';

class AlertBox extends StatelessWidget {
  final String title;
  final String description;
  final String imagePath;
  final Color backgroundColor;
  final Color borderColor;
  final Color textColor;

  const AlertBox({
    super.key,
    required this.title,
    required this.description,
    this.imagePath = 'assets/images/alert-danger.png',
    this.backgroundColor = AppColors.redlight,
    this.borderColor = AppColors.red,
    this.textColor = AppColors.red,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: backgroundColor,
        border: Border(left: BorderSide(width: 1, color: borderColor)),
      ),
      child: Row(
        children: [
          Image.asset(imagePath, height: 14),
          const SizedBox(width: 12),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: TextStyle(
                  fontSize: 10,
                  color: textColor,
                  fontWeight: FontWeight.w500,
                ),
              ),
              Text(
                description,
                style: TextStyle(
                  fontSize: 10,
                  color: textColor,
                  fontWeight: FontWeight.w300,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
