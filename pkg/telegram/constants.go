package telegram

const (
	RndCmd              = "/rnd"
	HelpCmd             = "/help"
	StartCmd            = "/start"
	StartWithPayloadCmd = "/start "
	SalesCmd            = "/sales"
	ReplenishCmd        = "/replenish"
	ContinueCmd         = "/continue"
)

const (
	CreateShopCmd  = "/create_shop"
	InviteUsersCmd = "/invite_user"
	ListUsersCmd   = "/list_users"
)

const (
	EditProductNameCmd        = "edit_product_name"
	EditProductCountCmd       = "edit_product_count"
	EditProductPurchaseCmd    = "edit_product_purchase"
	EditProductSellingCmd     = "edit_product_selling"
	EditProductDescriptionCmd = "edit_product_description"
	EditProductImagesCmd      = "edit_product_images"
	EditProductCmd            = "edit_product"
	ConfirmEditProductCmd     = "confirm_edit_product"
	DelProductCmd             = "del_product"
	ConfirmDelProductCmd      = "confirm_del_product"
	DelImageCmd               = "del_image"
	ConfirmDelImageCmd        = "confirm_del_image"
	FinishEditImagesCmd       = "finish_edit_images"
	ActionsProductCmd         = "actions_product"
	ActionsImagesCmd          = "actions_images"
	AddProductCmd             = "add_product"
	ListCmd                   = "list"
	AddItemToCartCmd          = "add_item_to_cart"
	ReduceItemInCartCmd       = "reduce_item_in_cart"
	RemoveItemFromCartCmd     = "remove_item_from_cart"
	EditCountItemInCartCmd    = "edit_count_item_in_cart"
	DiscountItemInCartCmd     = "discount_item_in_cart"
	SavePhoneNumberCmd        = "save_phone_number"
	DontSavePhoneNumberCmd    = "dont_save_phone_number"

	ShopKeyboardCmd        = "shop_keyboard"
	ReportsKeyboardCmd     = "reports_keyboad"
	ChangeShopCmd          = "change_shop"
	EditShopKeyboardCmd    = "edit_shop"
	DropShopCmd            = "drop_shop"
	UserListCmd            = "user_list"
	MenuCmd                = "menu"
	EditShopNameCmd        = "edit_shop_name"
	EditShopDescriptionCmd = "edit_shop_description"
	InviteEmployeeCmd      = "invite_employee"
	InviteClientCmd        = "invite_client"
	EditUserCmd            = "edit_user"
	GrantRoleCustomerCmd   = "grant_role_customer"
	GrantRoleEmployeeCmd   = "grant_role_employee"
	GrantRoleClientCmd     = "grant_role_client"
)

const (
	PayTypeCashCmd  = "pay_type_cash"
	PayTypeKaspiCmd = "pay_type_kaspi"
)

const (
	stateWaitingForPhoto         = 1
	stateWaitingForName          = 2
	stateWaitingForDescription   = 3
	stateWaitingForCount         = 4
	stateWaitingForPurchasePrice = 5
	stateWaitingForSellingPrice  = 6
)

const (
	stateWaitingForEditName          = 7
	stateWaitingForEditCount         = 8
	stateWaitingForEditPurchasePrice = 9
	stateWaitingForEditSellingPrice  = 10
	stateIdle                        = 11
	stateWaitingForEditImage         = 17
	stateWaitingForEditDescription   = 18
)

const (
	stateEditCountItemInCart   = 12
	stateDiscountProductInCart = 13
)

const (
	stateEditShopName        = 14
	stateEditShopDescription = 15
)

var addProductStates = map[int]bool{
	stateWaitingForPhoto:         true,
	stateWaitingForName:          true,
	stateWaitingForDescription:   true,
	stateWaitingForCount:         true,
	stateWaitingForPurchasePrice: true,
	stateWaitingForSellingPrice:  true,
}

var editProductStates = map[int]bool{
	stateWaitingForEditName:          true,
	stateWaitingForEditCount:         true,
	stateWaitingForEditPurchasePrice: true,
	stateWaitingForEditSellingPrice:  true,
	stateWaitingForEditImage:         true,
	stateWaitingForEditDescription:   true,
}

const (
	AddProductText       = "Добавить товар"
	CancelOperationsText = "Отмена"
	MenuText             = "Меню"
	PaymentText          = "Оплата"
)

const (
	roleCustomer = "customer" // Владелец магазина
	roleEmployee = "employee" // Сотрудник
	roleClient   = "client"   // Посетитель
)

const (
	stateWhatingClientPhone = 16
)
