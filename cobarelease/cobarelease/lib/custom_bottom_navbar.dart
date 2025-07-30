import 'package:flutter/material.dart';
import 'package:secpanel/theme/colors.dart';

class CustomBottomNavBar extends StatelessWidget {
  final int selectedIndex;
  final Function(int) onItemTapped;

  const CustomBottomNavBar({
    super.key,
    required this.selectedIndex,
    required this.onItemTapped,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceAround,
      children: [
        // Menggunakan label "Panel" sesuai nama aset
        _buildNavItem("Panel", 0),
        _buildNavItem("Profil", 1),
      ],
    );
  }

  Widget _buildNavItem(String label, int index) {
    final bool isSelected = selectedIndex == index;
    final color = isSelected ? AppColors.schneiderGreen : AppColors.gray;

    String imagePath;
    if (index == 0) {
      // Item pertama (Panel)
      imagePath = isSelected
          ? 'assets/images/panel-on.png'
          : 'assets/images/panel-off.png';
    } else {
      // Item kedua (Profile)
      imagePath = isSelected
          ? 'assets/images/profile-on.png'
          : 'assets/images/profile-off.png';
    }

    return Expanded(
      child: InkWell(
        onTap: () => onItemTapped(index),
        splashColor: Colors.transparent,
        highlightColor: Colors.transparent,
        hoverColor: Colors.transparent,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // ✨ Mengganti widget Icon dengan Image.asset
            Image.asset(
              imagePath,
              width: 24,
              height: 24,
              // Optional: Mencegah gambar diwarnai oleh tema
              color: null,
            ),
            const SizedBox(height: 4),
            Text(
              label,
              style: TextStyle(
                color: color,
                fontSize: 12,
                fontWeight: isSelected ? FontWeight.w500 : FontWeight.w400,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
