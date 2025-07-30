import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/paneldisplaydata.dart';
import 'package:secpanel/models/panels.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class EditPanelBottomSheet extends StatefulWidget {
  final PanelDisplayData panelData;
  final Company currentCompany;
  final List<Company> k3Vendors;
  final Function(Panel) onSave;
  final VoidCallback onDelete;

  const EditPanelBottomSheet({
    super.key,
    required this.panelData,
    required this.currentCompany,
    required this.k3Vendors,
    required this.onSave,
    required this.onDelete,
  });

  @override
  State<EditPanelBottomSheet> createState() => _EditPanelBottomSheetState();
}

class _EditPanelBottomSheetState extends State<EditPanelBottomSheet> {
  final _formKey = GlobalKey<FormState>();

  // Controllers for text fields
  late final TextEditingController _noPanelController;
  late final TextEditingController _noWbsController;
  late final TextEditingController _noPpController;
  late final TextEditingController _progressController;

  // State variables
  late Panel _panel; // A local, mutable copy of the panel
  late DateTime _selectedDate;
  late String? _selectedK3VendorId;
  late bool _isSent; // State for "Tandai Sudah Dikirim"
  late bool _canMarkAsSent;

  bool _isLoading = false;
  bool _isSuccess = false;

  // Role checks
  bool get _isAdmin => widget.currentCompany.role == AppRole.admin;
  bool get _isK3 => widget.currentCompany.role == AppRole.k3;

  @override
  void initState() {
    super.initState();
    // Create a deep copy to avoid modifying the original panel data directly
    _panel = Panel.fromMap(widget.panelData.panel.toMap());

    // Initialize controllers and state from the panel data
    _noPanelController = TextEditingController(text: _panel.noPanel);
    _noWbsController = TextEditingController(text: _panel.noWbs);
    _noPpController = TextEditingController(text: _panel.noPp);
    _progressController = TextEditingController(
      text: _panel.percentProgress?.toInt().toString() ?? '0',
    );
    _selectedDate = _panel.startDate ?? DateTime.now();
    _selectedK3VendorId = _panel.vendorId;
    _isSent = _panel.isClosed;

    // Determine initial state for the "Mark as Sent" checkbox
    _updateCanMarkAsSent();

    // Listen for changes in the progress field to enable/disable the checkbox
    _progressController.addListener(_updateCanMarkAsSent);
  }

  @override
  void dispose() {
    _noPanelController.dispose();
    _noWbsController.dispose();
    _noPpController.dispose();
    _progressController.removeListener(_updateCanMarkAsSent);
    _progressController.dispose();
    super.dispose();
  }

  void _updateCanMarkAsSent() {
    final progress = int.tryParse(_progressController.text) ?? 0;
    if (mounted) {
      setState(() {
        _canMarkAsSent = progress >= 75;
        // If progress drops below 75, uncheck and disable "Mark as Sent"
        if (!_canMarkAsSent) {
          _isSent = false;
        }
      });
    }
  }

  Future<void> _saveChanges() async {
    if (_isLoading || _isSuccess) return;
    if (_formKey.currentState!.validate()) {
      if (_isAdmin && _selectedK3VendorId == null) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Admin harus memilih Vendor Panel (K3)'),
            backgroundColor: Colors.red,
          ),
        );
        return;
      }

      setState(() => _isLoading = true);
      final noPanel = _noPanelController.text.trim();
      final originalNoPp = widget.panelData.panel.noPp;

      final isUnique = await DatabaseHelper.instance.isPanelNumberUnique(
        noPanel,
        currentNoPp: originalNoPp, // Kirim No. PP asli untuk pengecualian
      );
      if (!isUnique) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('No. Panel sudah digunakan oleh panel lain.'),
              backgroundColor: Colors.red,
            ),
          );
          setState(() => _isLoading = false);
        }
        return;
      }
      // Update the local panel object with the new values
      _panel.noPanel = _noPanelController.text.trim();
      _panel.noWbs = _noWbsController.text.trim();
      _panel.noPp = _noPpController.text.trim();
      _panel.percentProgress =
          double.tryParse(_progressController.text.trim()) ?? 0.0;
      _panel.startDate = _selectedDate;
      _panel.vendorId = _selectedK3VendorId;
      _panel.isClosed = _isSent;

      // Update closed date based on the "isSent" status
      if (_isSent && _panel.closedDate == null) {
        _panel.closedDate = DateTime.now();
      } else if (!_isSent) {
        _panel.closedDate = null;
      }

      try {
        await DatabaseHelper.instance.updatePanel(_panel);
        setState(() {
          _isLoading = false;
          _isSuccess = true;
        });
        await Future.delayed(const Duration(milliseconds: 1500));

        if (mounted) {
          widget.onSave(_panel); // Pass the updated panel back
          Navigator.pop(context);
        }
      } catch (e) {
        if (mounted) {
          setState(() => _isLoading = false);
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text('Gagal menyimpan: ${e.toString()}'),
              backgroundColor: Colors.red,
            ),
          );
        }
      }
    }
  }

  void _showDeleteConfirmation() {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (modalContext) => Padding(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              "Hapus Panel?",
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),
            Text(
              "Anda yakin ingin menghapus panel ${_panel.noPanel}? Tindakan ini tidak dapat diurungkan.",
              style: const TextStyle(color: AppColors.gray, fontSize: 14),
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => Navigator.pop(modalContext),
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      side: const BorderSide(color: AppColors.grayLight),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                    child: const Text(
                      "Batal",
                      style: TextStyle(color: AppColors.black, fontSize: 12),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: ElevatedButton(
                    onPressed: () {
                      Navigator.pop(modalContext);
                      widget
                          .onDelete(); // This will trigger deletion in the home screen
                    },
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      backgroundColor: AppColors.red,
                      elevation: 0,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6),
                      ),
                    ),
                    child: const Text(
                      "Ya, Hapus",
                      style: TextStyle(color: AppColors.white, fontSize: 12),
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SingleChildScrollView(
        padding: EdgeInsets.only(
          left: 20,
          right: 20,
          top: 16,
          bottom: MediaQuery.of(context).viewInsets.bottom + 16,
        ),
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Drag handle
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

              // Header
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  // Title
                  Text(
                    "Edit Panel ${_panel.noPanel}",
                    style: const TextStyle(
                      fontSize: 20,
                      fontWeight: FontWeight.w500,
                    ),
                  ),

                  // -- ✨ CHANGE IS HERE --
                  // Only show the delete icon if the user is an admin
                  if (_isAdmin)
                    IconButton(
                      icon: const Icon(
                        Icons.delete_outline,
                        color: AppColors.red,
                      ),
                      onPressed: _showDeleteConfirmation,
                    ),
                ],
              ),
              const SizedBox(height: 16),

              // "Tandai Sudah Dikirim" widget
              if (_isAdmin || _isK3) ...[
                _buildMarkAsSent(),
                const SizedBox(height: 16),
              ],

              // Form Fields
              _buildTextField(
                controller: _noPanelController,
                label: "No. Panel",
              ),
              const SizedBox(height: 16),
              _buildTextField(controller: _noWbsController, label: "No. WBS"),
              const SizedBox(height: 16),
              _buildTextField(controller: _noPpController, label: "No. PP"),
              const SizedBox(height: 16),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    flex: 1,
                    child: _buildTextField(
                      controller: _progressController,
                      label: "Progress",
                      isNumber: true,
                      suffixText: "%",
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(flex: 2, child: _buildDateTimePicker()),
                ],
              ),
              const SizedBox(height: 16),
              if (_isAdmin)
                _buildAdminVendorPicker()
              else if (_isK3)
                _buildK3VendorDisplay(),
              const SizedBox(height: 32),
              _buildActionButtons(),
            ],
          ),
        ),
      ),
    );
  }

  // --- WIDGET BUILDER HELPERS ---

  Widget _buildMarkAsSent() {
    final Color borderColor = _isSent
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    final Color bgColor = _isSent
        ? AppColors.schneiderGreen.withOpacity(0.08)
        : AppColors.white;
    final Color textColor = _canMarkAsSent ? AppColors.black : AppColors.gray;

    return InkWell(
      onTap: _canMarkAsSent ? () => setState(() => _isSent = !_isSent) : null,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: bgColor,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              Icons.local_shipping_outlined,
              color: _canMarkAsSent ? AppColors.schneiderGreen : AppColors.gray,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    "Tandai Sudah Dikirim",
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      color: textColor,
                    ),
                  ),
                  Text(
                    "Bisa diklik jika progres terisi di atas 75%",
                    style: TextStyle(
                      fontSize: 10,
                      color: textColor,
                      fontWeight: FontWeight.w300,
                    ),
                  ),
                ],
              ),
            ),
            Checkbox(
              value: _isSent,
              onChanged: _canMarkAsSent
                  ? (bool? value) {
                      setState(() {
                        _isSent = value ?? false;
                      });
                    }
                  : null, // Disables the checkbox
              activeColor: AppColors.schneiderGreen,
              checkColor: Colors.white,
              visualDensity: VisualDensity.compact,
              side: BorderSide(
                color: _canMarkAsSent ? AppColors.gray : AppColors.grayLight,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String label,
    bool isNumber = false,
    String? suffixText,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        TextFormField(
          cursorColor: AppColors.schneiderGreen,
          controller: controller,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
          keyboardType: isNumber ? TextInputType.number : TextInputType.text,
          validator: (value) {
            if (isNumber) {
              if (value == null || value.isEmpty) {
                return '0-100';
              }
              final progress = int.tryParse(value);
              if (progress == null) {
                return '0-100';
              }
              if (progress < 0 || progress > 100) {
                return '0-100';
              }
            }
            // Validator untuk field non-angka
            if (!isNumber && (value == null || value.isEmpty)) {
              return 'Field ini tidak boleh kosong';
            }
            return null; // Lolos validasi
          },

          decoration: InputDecoration(
            suffixText: suffixText,
            hintText: 'Masukkan $label',
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 12,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.schneiderGreen),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildDateTimePicker() {
    Future<void> pickDateTime() async {
      final date = await showDatePicker(
        context: context,
        initialDate: _selectedDate,
        firstDate: DateTime(2000),
        lastDate: DateTime(2101),
        initialEntryMode: DatePickerEntryMode.calendarOnly,
        builder: (context, child) {
          return Theme(
            data: ThemeData.light().copyWith(
              colorScheme: const ColorScheme.light(
                primary: AppColors.schneiderGreen,
                onPrimary: Colors.white,
                onSurface: AppColors.black,
              ),
              textButtonTheme: TextButtonThemeData(
                style: ButtonStyle(
                  foregroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return Colors.white;
                    }
                    return AppColors.schneiderGreen;
                  }),
                  backgroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return AppColors.schneiderGreen;
                    }
                    return Colors.transparent;
                  }),
                  side: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return BorderSide.none;
                    }
                    return const BorderSide(color: AppColors.schneiderGreen);
                  }),
                  padding: WidgetStateProperty.all(
                    const EdgeInsets.symmetric(vertical: 12, horizontal: 24),
                  ),
                  shape: WidgetStateProperty.all(
                    RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  textStyle: WidgetStateProperty.all(
                    const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      fontFamily: 'Lexend',
                    ),
                  ),
                ),
              ),
              datePickerTheme: DatePickerThemeData(
                backgroundColor: Colors.white,
                headerBackgroundColor: AppColors.white,
                headerForegroundColor: AppColors.black,
                dividerColor: AppColors.grayLight,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
            ),
            child: child!,
          );
        },
      );
      if (date == null) return;

      final time = await showTimePicker(
        context: context,
        initialTime: TimeOfDay.fromDateTime(_selectedDate),
        builder: (context, child) {
          return Theme(
            data: ThemeData.light().copyWith(
              colorScheme: const ColorScheme.light(
                primary: AppColors.schneiderGreen,
                onPrimary: Colors.white,
                onSurface: Colors.black,
                surface: Colors.white,
              ),
              textButtonTheme: TextButtonThemeData(
                style: ButtonStyle(
                  foregroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return Colors.white;
                    }
                    return AppColors.schneiderGreen;
                  }),
                  backgroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return AppColors.schneiderGreen;
                    }
                    return Colors.transparent;
                  }),
                  side: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return BorderSide.none;
                    }
                    return const BorderSide(color: AppColors.schneiderGreen);
                  }),
                  padding: WidgetStateProperty.all(
                    const EdgeInsets.symmetric(vertical: 12, horizontal: 24),
                  ),
                  shape: WidgetStateProperty.all(
                    RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  textStyle: WidgetStateProperty.all(
                    const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      fontFamily: 'Lexend',
                    ),
                  ),
                ),
              ),
            ),
            child: child!,
          );
        },
      );
      if (time == null) return;

      setState(() {
        _selectedDate = DateTime(
          date.year,
          date.month,
          date.day,
          time.hour,
          time.minute,
        );
      });
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Waktu Mulai Pengerjaan",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        InkWell(
          onTap: pickDateTime,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.grayLight),
            ),
            child: Row(
              children: [
                const Icon(
                  Icons.calendar_today_outlined,
                  size: 20,
                  color: AppColors.gray,
                ),
                const SizedBox(width: 8),
                Text(
                  DateFormat('d MMM yyyy HH:mm').format(_selectedDate),
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w300,
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildAdminVendorPicker() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Panel",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 12,
          children: widget.k3Vendors.map((vendor) {
            return _buildOptionButton(
              label: vendor.name,
              selected: _selectedK3VendorId == vendor.id,
              onTap: () => setState(() => _selectedK3VendorId = vendor.id),
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _buildK3VendorDisplay() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Panel",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: AppColors.grayLight.withOpacity(0.5),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: AppColors.grayLight),
          ),
          child: Text(
            widget.currentCompany.name,
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w300,
              color: AppColors.gray,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildOptionButton({
    required String label,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: selected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: const TextStyle(fontWeight: FontWeight.w400, fontSize: 12),
        ),
      ),
    );
  }

  Widget _buildActionButtons() {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: () => Navigator.pop(context),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              side: const BorderSide(color: AppColors.schneiderGreen),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text(
              "Batal",
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: _saveChanges,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: _isSuccess
                  ? Colors.green
                  : AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: _isLoading
                ? const SizedBox(
                    height: 16,
                    width: 16,
                    child: CircularProgressIndicator(
                      color: Colors.white,
                      strokeWidth: 2,
                    ),
                  )
                : _isSuccess
                ? const Icon(Icons.check, size: 16)
                : const Text("Simpan", style: TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }
}
